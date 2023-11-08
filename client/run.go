package client

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/codingpa-ws/foxbox/internal/store"

	seccomp "github.com/seccomp/libseccomp-golang"
	"golang.org/x/sys/unix"
)

type RunOptions struct {
	Command []string
	Store   *store.Store
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

	err = run(name, entry.FileSystem(), opt.Command)

	return
}

func run(name string, dir string, command []string) error {
	cmd := exec.Command("/proc/self/exe", command...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Dir = dir
	cmd.Env = []string{"FOXBOX_EXEC=" + name}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWPID | syscall.CLONE_NEWNET | syscall.CLONE_NEWTIME,
		Unshareflags: syscall.CLONE_NEWNS,
		UidMappings: []syscall.SysProcIDMap{{
			ContainerID: 0,
			HostID:      1000,
			Size:        1}},
	}
	err := cmd.Run()
	if cmd.ProcessState == nil {
		return err
	}
	exit := cmd.ProcessState.ExitCode()
	if exit != 0 {
		os.Exit(exit)
	}
	return nil
}
func child() (err error) {
	err = prepareFs()
	if err != nil {
		return err
	}
	name := os.Getenv("FOXBOX_EXEC")
	err = syscall.Sethostname([]byte(name))
	if err != nil {
		return fmt.Errorf("setting hostname to %s: %w", name, err)
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
	fmt.Println("ENDED")
	return nil
}

func prepareFs() (err error) {
	return errors.Join(
		syscall.Chroot("."),
		os.Chdir("/"),
		syscall.Mount("proc", "proc", "proc", 0, ""),
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
