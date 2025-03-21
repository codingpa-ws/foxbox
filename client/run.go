package client

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/codingpa-ws/foxbox/internal/cgroup2"
	"github.com/codingpa-ws/foxbox/internal/security"
	"github.com/codingpa-ws/foxbox/internal/slirp"
	"github.com/codingpa-ws/foxbox/internal/store"

	"golang.org/x/sys/unix"
)

type VolumeConfig struct {
	HostPath, BoxPath string
}

type RunOptions struct {
	Command []string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	Volumes []VolumeConfig

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
func (self RunOptions) NeedsCGroup() bool {
	return self.MaxCPUs > 0 || self.MaxMemoryBytes > 0 || self.MaxProcesses > 0
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

func (client *client) Run(name string, opt *RunOptions) (err error) {
	opt = newOr(opt)

	entry, err := client.store.GetEntry(name)
	if err != nil {
		return
	}

	err = run(name, entry, opt)

	return
}

func run(name string, entry *store.StoreEntry, opt *RunOptions) error {
	conflictingPID, running, err := entry.GetPID()
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("getting pid of potentially conflicting process: %w", err)
	}
	if running {
		return fmt.Errorf("already running with pid %d, use client.Exec", conflictingPID)
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding foxbox executable: %w", err)
	}

	cgroup, err := cgroup2.Open("foxbox-" + name)
	if err != nil {
		return fmt.Errorf("creating cgroup foxbox-%s: %w", name, err)
	}
	defer func() {
		err = errors.Join(err, cgroup.Delete())
	}()

	var useCGroup = opt.NeedsCGroup() && os.Getenv("CI_NO_CGROUP") == ""
	var cgroupFd int

	if useCGroup {
		cgroupDir, err := os.Open(cgroup.Path())
		if err != nil {
			return fmt.Errorf("opening cgroup dir: %w", err)
		}
		defer cgroupDir.Close()
		err = setupCgroup(cgroup, opt)
		if err != nil {
			return fmt.Errorf("setting up cgroup: %w", err)
		}
		cgroupFd = int(cgroupDir.Fd())
	}
	sysProcAttr, err := security.GetSysProcAttr(cgroupFd, useCGroup)
	if err != nil {
		return fmt.Errorf("getting proc attributes: %w", err)
	}

	volumes, err := encodeVolumes(opt.Volumes)
	if err != nil {
		return fmt.Errorf("encoding volume data (%v): %w", opt.Volumes, err)
	}

	noTmpfs := "0"
	if opt.MaxMemoryBytes > 0 {
		noTmpfs = "1"
	}

	cmd := exec.Command(executable, opt.Command...)
	cmd.Stdin = opt.getStdin()
	cmd.Stdout = opt.getStdout()
	cmd.Stderr = opt.getStderr()
	cmd.Dir = entry.FileSystem()
	cmd.Env = []string{"FOXBOX_EXEC=" + name, "FOXBOX_MOUNTS=" + volumes, "FOXBOX_NO_TMPFS=" + noTmpfs}
	cmd.SysProcAttr = sysProcAttr

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("starting process: %w", err)
	}
	err = entry.SetPID(cmd.Process.Pid)
	if err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("setting box pid: %w", err)
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

func encodeVolumes(volumes []VolumeConfig) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting work dir: %w", err)
	}
	for key, volume := range volumes {
		if !filepath.IsAbs(volume.HostPath) {
			volume.HostPath = filepath.Join(wd, volume.HostPath)
		}

		if !filepath.IsAbs(volume.BoxPath) {
			volume.BoxPath = "/" + volume.BoxPath
		}
		volumes[key] = volume
	}

	mountBuf := new(bytes.Buffer)
	err = gob.NewEncoder(mountBuf).Encode(volumes)
	if err != nil {
		return "", fmt.Errorf("serializing volumes: %w", err)
	}

	return fmt.Sprintf("%x", mountBuf.String()), nil
}

func child() (err error) {
	name := os.Getenv("FOXBOX_EXEC")
	err = prepareFs()
	if err != nil {
		return err
	}
	err = syscall.Sethostname([]byte(name))
	if err != nil {
		return fmt.Errorf("setting hostname to %s: %w", name, err)
	}
	_, err = os.ReadFile("/etc/hostname")
	if !os.IsNotExist(err) {
		_ = os.WriteFile("/etc/hostname", []byte(name+"\n"), 0644)
	}
	err = linkStandardStreams()
	if err != nil {
		return
	}
	err = security.DropCapabilities()
	if err != nil {
		return fmt.Errorf("dropping capabilities: %w", err)
	}
	err = security.RestrictSyscalls()
	if err != nil {
		return fmt.Errorf("restricting syscalls: %w", err)
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

func prepareFs() (err error) {
	volumes, err := decodeVolumeMounts()
	if err != nil {
		return
	}

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

	for _, volume := range volumes {
		err = os.MkdirAll("."+volume.BoxPath, 0777)
		if err != nil && !os.IsExist(err) {
			return fmt.Errorf("preparing volume %s: %w", volume.BoxPath, err)
		}
		err = unix.Mount(volume.HostPath, "."+volume.BoxPath, "", unix.MS_BIND|unix.MS_REC, "")
		if err != nil {
			return fmt.Errorf("mounting %s: %w", volume.BoxPath, err)
		}
	}

	return enterFs()
}

func decodeVolumeMounts() (volumes []VolumeConfig, err error) {
	var b []byte
	_, err = fmt.Sscanf(os.Getenv("FOXBOX_MOUNTS"), "%x", &b)
	if err != nil {
		return nil, fmt.Errorf("reading FOXBOX_MOUNTS: %w", err)
	}
	err = gob.NewDecoder(bytes.NewReader(b)).Decode(&volumes)
	if err != nil {
		return nil, fmt.Errorf("decoding volume config: %w", err)
	}
	return
}

func enterFs() (err error) {
	err = syscall.Chroot(".")
	if err != nil {
		return
	}
	err = os.Chdir("/")
	if err != nil {
		return
	}
	err = syscall.Mount("proc", "proc", "proc", 0, "")
	if err != nil {
		return
	}
	if os.Getenv("FOXBOX_NO_TMPFS") != "1" {
		err = syscall.Mount("tmpfs", "tmp", "tmpfs", 0, "")
		if err != nil {
			return
		}
	}
	// TODO: figure out if sysfs can be mounted securely?
	// syscall.Mount("sysfs", "sys", "sysfs", 0, ""),
	return
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
