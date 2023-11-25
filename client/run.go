package client

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"

	"github.com/codingpa-ws/foxbox/internal/cgroup2"
	"github.com/codingpa-ws/foxbox/internal/slirp"
	"github.com/codingpa-ws/foxbox/internal/store"

	seccomp "github.com/seccomp/libseccomp-golang"
	"golang.org/x/sys/unix"
)

type RunOptions struct {
	Command []string
	Store   *store.Store

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	EnableNetworking bool

	MaxCPUs        float32
	MaxMemoryBytes uint
	MaxProcesses   uint
}

func (self RunOptions) getStdin() io.Reader {
	if self.Stdin == nil {
		return os.Stdin
	}
	return self.Stdin
}
func (self RunOptions) getStdout() io.Writer {
	if self.Stdout == nil {
		return os.Stdout
	}
	return self.Stdout
}

func (self RunOptions) getStderr() io.Writer {
	if self.Stderr == nil {
		return os.Stderr
	}
	return self.Stderr
}

func init() {
	if _, ok := os.LookupEnv("FOXBOX_EXEC"); ok {
		err := child()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return
	}
}

func Run(name string, opt *RunOptions) (err error) {
	opt = newOr(opt)

	if opt.Store == nil {
		opt.Store, err = store.New("runtime")
		if err != nil {
			return
		}
	}

	entry, err := opt.Store.GetEntry(name)
	if err != nil {
		return
	}

	err = run(name, entry.FileSystem(), opt)

	return
}

func run(name string, dir string, opt *RunOptions) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding foxbox executable: %w", err)
	}

	uid, gid, err := getUserIdentifiers()
	if err != nil {
		return err
	}
	cgroup, err := cgroup2.Open("foxbox-" + name)
	if err != nil {
		return fmt.Errorf("creating cgroup foxbox-%s: %w", name, err)
	}
	defer func() {
		err = errors.Join(err, cgroup.Delete())
	}()

	cgroupDir, err := os.Open(cgroup.Path())
	if err != nil {
		return fmt.Errorf("opening cgroup dir: %w", err)
	}
	defer cgroupDir.Close()
	err = setupCgroup(cgroup, opt)
	if err != nil {
		return fmt.Errorf("setting up cgroup: %w", err)
	}

	cmd := exec.Command(executable, opt.Command...)
	cmd.Stdin = opt.getStdin()
	cmd.Stdout = opt.getStdout()
	cmd.Stderr = opt.getStderr()
	cmd.Dir = dir
	cmd.Env = []string{"FOXBOX_EXEC=" + name}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWPID | syscall.CLONE_NEWNET | syscall.CLONE_NEWTIME | syscall.CLONE_NEWCGROUP,
		Unshareflags: syscall.CLONE_NEWNS,
		UidMappings: []syscall.SysProcIDMap{{
			ContainerID: 0,
			HostID:      int(uid),
			Size:        1}},
		GidMappings: []syscall.SysProcIDMap{{
			ContainerID: 0,
			HostID:      int(gid),
			Size:        1,
		}},
		CgroupFD:    int(cgroupDir.Fd()),
		UseCgroupFD: os.Getenv("CI_NO_CGROUP") == "",
	}
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("starting process: %w", err)
	}
	if opt.EnableNetworking {
		slirp, err := slirp.Start(cmd.Process.Pid)
		if err != nil {
			cmd.Process.Kill()
			return fmt.Errorf("starting slirp (networking): %w", err)
		}
		defer slirp.Process.Kill()
	}
	err = cmd.Wait()
	if cmd.ProcessState == nil {
		return fmt.Errorf("starting process (no process state): %w", err)
	}
	return err
}

func getUserIdentifiers() (uid, gid int, err error) {
	user, err := user.Current()
	if err != nil {
		return 0, 0, fmt.Errorf("getting current user: %w", err)
	}

	uid, err = strconv.Atoi(user.Uid)
	if err != nil {
		return 0, 0, fmt.Errorf("parsing user uid (%s): %w", user.Uid, err)
	}
	gid, err = strconv.Atoi(user.Gid)
	if err != nil {
		return 0, 0, fmt.Errorf("parsing user gid (%s): %w", user.Gid, err)
	}

	return
}

func setupCgroup(cgroup *cgroup2.CGroup, opt *RunOptions) (err error) {
	var maxProcesses uint = 10_000
	if opt.MaxProcesses > 0 {
		maxProcesses = opt.MaxProcesses
	}

	err = cgroup.LimitPIDs(maxProcesses)
	if err != nil {
		return fmt.Errorf("limiting pid count: %w", err)
	}

	if opt.MaxCPUs > 0 {
		err = cgroup.LimitCPUs(opt.MaxCPUs)
		if err != nil {
			return fmt.Errorf("limiting cpu count: %w", err)
		}
	}

	if opt.MaxMemoryBytes > 0 {
		err = cgroup.LimitMemory(opt.MaxMemoryBytes)
	}

	return
}

func child() (err error) {
	name := os.Getenv("FOXBOX_EXEC")
	err = prepareFs(name)
	if err != nil {
		return err
	}
	err = syscall.Sethostname([]byte(name))
	if err != nil {
		return fmt.Errorf("setting hostname to %s: %w", name, err)
	}
	err = linkStandardStreams()
	if err != nil {
		return
	}
	err = dropCapabilities()
	if err != nil {
		return fmt.Errorf("dropping capabilities: %w", err)
	}
	err = dropSyscalls()
	if err != nil {
		return fmt.Errorf("dropping syscall permissions: %w", err)
	}
	args := []string{"sh"}
	if len(os.Args) > 1 {
		args = os.Args[1:]
	}

	defer syscall.Unmount("proc", 0)
	err = syscall.Exec("/bin/sh", args, []string{"PATH=/bin:/sbin:/usr/bin:/usr/sbin", "LANG=C.UTF-8", "CHARSET=UTF-8"})
	if err != nil {
		return err
	}
	return nil
}

func prepareFs(name string) (err error) {
	devices := []string{
		"/dev/null",
		"/dev/zero",
		"/dev/full",
		"/dev/tty",
		"/dev/random",
		"/dev/urandom",
	}

	for _, device := range devices {
		err = os.WriteFile("."+device, []byte{}, 0666)
		if err != nil {
			return fmt.Errorf("creating %s: %w", device, err)
		}
		err = unix.Mount(device, "."+device, "", unix.MS_BIND|unix.MS_REC, "")
		if err != nil {
			return fmt.Errorf("mounting %s: %w", device, err)
		}
	}

	return enterFs()
}

func enterFs() (err error) {
	return errors.Join(
		syscall.Chroot("."),
		os.Chdir("/"),
		syscall.Mount("proc", "proc", "proc", 0, ""),
		// TODO: figure out if sysfs can be mounted securely?
		// syscall.Mount("sysfs", "sys", "sysfs", 0, ""),
	)
}

func dropSyscalls() error {
	var SCMP_FAIL = seccomp.ActErrno.SetReturnCode(int16(unix.EPERM))
	filter, err := seccomp.NewFilter(seccomp.ActAllow)
	if err != nil {
		return fmt.Errorf("creating seccomp filter: %w", err)
	}
	defer filter.Release()

	name := "chmod"
	call, err := seccomp.GetSyscallFromName(name)
	if err != nil {
		return fmt.Errorf("getting syscall `%s`: %w", name, err)
	}
	err = filter.AddRuleConditional(call, SCMP_FAIL, []seccomp.ScmpCondition{
		{
			Argument: 1,
			Operand1: syscall.S_ISUID,
			Operand2: syscall.S_ISUID,
			Op:       seccomp.CompareMaskedEqual,
		},
	})
	if err != nil {
		return fmt.Errorf("adding rule for %s: %w", name, err)
	}
	err = filter.AddRuleConditional(call, SCMP_FAIL, []seccomp.ScmpCondition{
		{
			Argument: 1,
			Operand1: syscall.S_ISGID,
			Operand2: syscall.S_ISGID,
			Op:       seccomp.CompareMaskedEqual,
		},
	})
	if err != nil {
		return fmt.Errorf("adding rule for %s: %w", name, err)
	}
	name = "fchmod"
	call, err = seccomp.GetSyscallFromName(name)
	if err != nil {
		return fmt.Errorf("getting syscall `%s`: %w", name, err)
	}
	err = filter.AddRuleConditional(call, SCMP_FAIL, []seccomp.ScmpCondition{
		{
			Argument: 1,
			Operand1: syscall.S_ISUID,
			Operand2: syscall.S_ISUID,
			Op:       seccomp.CompareMaskedEqual,
		},
	})
	if err != nil {
		return fmt.Errorf("adding rule for %s: %w", name, err)
	}
	err = filter.AddRuleConditional(call, SCMP_FAIL, []seccomp.ScmpCondition{
		{
			Argument: 1,
			Operand1: syscall.S_ISGID,
			Operand2: syscall.S_ISGID,
			Op:       seccomp.CompareMaskedEqual,
		},
	})
	if err != nil {
		return fmt.Errorf("adding rule for %s: %w", name, err)
	}

	name = "fchmodat"
	call, err = seccomp.GetSyscallFromName(name)
	if err != nil {
		return fmt.Errorf("getting syscall `%s`: %w", name, err)
	}
	err = filter.AddRuleConditional(call, SCMP_FAIL, []seccomp.ScmpCondition{
		{
			Argument: 2,
			Operand1: syscall.S_ISUID,
			Operand2: syscall.S_ISUID,
			Op:       seccomp.CompareMaskedEqual,
		},
	})
	if err != nil {
		return fmt.Errorf("adding rule for %s: %w", name, err)
	}
	err = filter.AddRuleConditional(call, SCMP_FAIL, []seccomp.ScmpCondition{
		{
			Argument: 2,
			Operand1: syscall.S_ISGID,
			Operand2: syscall.S_ISGID,
			Op:       seccomp.CompareMaskedEqual,
		},
	})
	if err != nil {
		return fmt.Errorf("adding rule for %s: %w", name, err)
	}

	name = "unshare"
	call, err = seccomp.GetSyscallFromName("unshare")
	if err != nil {
		return fmt.Errorf("getting syscall `%s`: %w", name, err)
	}
	err = filter.AddRuleConditional(call, SCMP_FAIL, []seccomp.ScmpCondition{
		{
			Argument: 0,
			Operand1: syscall.CLONE_NEWUSER,
			Operand2: syscall.CLONE_NEWUSER,
			Op:       seccomp.CompareMaskedEqual,
		},
	})
	if err != nil {
		return fmt.Errorf("adding rule for %s: %w", name, err)
	}

	name = "clone"
	call, err = seccomp.GetSyscallFromName(name)
	if err != nil {
		return fmt.Errorf("getting syscall `%s`: %w", name, err)
	}
	err = filter.AddRuleConditional(call, SCMP_FAIL, []seccomp.ScmpCondition{
		{
			Argument: 0,
			Operand1: syscall.CLONE_NEWUSER,
			Operand2: syscall.CLONE_NEWUSER,
			Op:       seccomp.CompareMaskedEqual,
		},
	})
	if err != nil {
		return fmt.Errorf("adding rule for %s: %w", name, err)
	}

	name = "ioctl"
	call, err = seccomp.GetSyscallFromName(name)
	if err != nil {
		return fmt.Errorf("getting syscall `%s`: %w", name, err)
	}
	err = filter.AddRuleConditional(call, SCMP_FAIL, []seccomp.ScmpCondition{
		{
			Argument: 0,
			Operand1: syscall.CLONE_NEWUSER,
			Operand2: syscall.CLONE_NEWUSER,
			Op:       seccomp.CompareMaskedEqual,
		},
	})
	if err != nil {
		return fmt.Errorf("adding rule for %s: %w", name, err)
	}

	err = filter.SetNoNewPrivsBit(false)
	if err != nil {
		return fmt.Errorf("setting SCMP_FLTATR_CTL_NNP: %w", err)
	}

	forbiddenCalls := []string{
		"keyctl",
		"add_key",
		"request_key",
		"ptrace",
		"mbind",
		"migrate_pages",
		"move_pages",
		"set_mempolicy",
		"userfaultfd",
		"perf_event_open",
		"chroot",
	}

	for _, name := range forbiddenCalls {
		call, err := seccomp.GetSyscallFromName(name)

		if err != nil {
			return fmt.Errorf("getting syscall `%s`: %w", name, err)
		}
		err = filter.AddRule(call, SCMP_FAIL)

		if err != nil {
			return fmt.Errorf("adding rule for `%s`: %w", name, err)
		}
	}

	return filter.Load()
}

func dropCapabilities() error {
	caps := []uintptr{
		unix.CAP_AUDIT_CONTROL,
		unix.CAP_AUDIT_READ,
		unix.CAP_AUDIT_WRITE,
		unix.CAP_BLOCK_SUSPEND,
		unix.CAP_DAC_READ_SEARCH,
		unix.CAP_FSETID,
		unix.CAP_IPC_LOCK,
		unix.CAP_MAC_ADMIN,
		unix.CAP_MAC_OVERRIDE,
		unix.CAP_MKNOD,
		unix.CAP_SETFCAP,
		unix.CAP_SYSLOG,
		unix.CAP_SYS_ADMIN,
		unix.CAP_SYS_BOOT,
		unix.CAP_SYS_MODULE,
		unix.CAP_SYS_NICE,
		unix.CAP_SYS_RAWIO,
		unix.CAP_SYS_RESOURCE,
		unix.CAP_SYS_TIME,
		unix.CAP_WAKE_ALARM,
	}

	for _, cap := range caps {
		err := unix.Prctl(unix.PR_CAPBSET_DROP, cap, 0, 0, 0)

		if err != nil {
			return fmt.Errorf("dropping capability %#x: %w", cap, err)
		}
	}

	return nil
}

func linkStandardStreams() (err error) {
	// Removing can be silent because we’ll fail
	// with EPERM etc. in symlink below anyway
	// TODO: move this to client.Create() perhaps?
	_ = os.Remove("/dev/stdin")
	_ = os.Remove("/dev/stdout")
	_ = os.Remove("/dev/stderr")
	_ = os.Remove("/dev/fd")

	err = syscall.Symlink("/proc/self/fd/0", "/dev/stdin")
	if err != nil {
		return fmt.Errorf("linking /dev/stdin: %w", err)
	}
	err = syscall.Symlink("/proc/self/fd/1", "/dev/stdout")
	if err != nil {
		return fmt.Errorf("linking /dev/stdout: %w", err)
	}
	err = syscall.Symlink("/proc/self/fd/2", "/dev/stderr")
	if err != nil {
		return fmt.Errorf("linking /dev/stderr: %w", err)
	}
	err = syscall.Symlink("/proc/self/fd", "/dev/fd")
	if err != nil {
		return fmt.Errorf("linking /dev/fd: %w", err)
	}
	return
}
