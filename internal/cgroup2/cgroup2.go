package cgroup2

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

type CGroup struct {
	path string
}

func (self CGroup) Name() string {
	return filepath.Base(self.path)
}

func (self CGroup) Path() string {
	return self.path
}

func (self CGroup) Delete() error {
	return syscall.Rmdir(self.Path())
}

func (self CGroup) write(file string, value string) error {
	path := filepath.Join(self.Path(), sanitize(file))
	return os.WriteFile(path, []byte(value), 0)
}

func (self CGroup) LimitPIDs(maxPIDs uint) error {
	return self.write("pids.max", strconv.Itoa(int(maxPIDs)))
}

func (self CGroup) LimitMemory(maxBytes uint) error {
	return self.write("memory.max", strconv.Itoa(int(maxBytes)))
}

func (self CGroup) LimitCPUs(cores float32) error {
	const baseMicroseconds float32 = 100000

	max := int(baseMicroseconds * cores)

	return self.write("cpu.max", fmt.Sprintf("%d %.0f", max, baseMicroseconds))
}

func (self CGroup) AddPID(pid int) error {
	f, err := os.OpenFile(filepath.Join(self.path, "cgroup.procs"), os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	_, err = f.Write([]byte(strconv.Itoa(pid)))
	if err != nil {
		return err
	}
	return f.Close()
}

func Open(name string) (*CGroup, error) {
	uid := os.Getuid()
	const basePath = "/sys/fs/cgroup/user.slice/"
	path := fmt.Sprintf("%suser-%d.slice/user@%d.service/app.slice/%s", basePath, uid, uid, name)

	cgroup := FromPath(path)

	err := os.Mkdir(path, 0777)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("creating cgroup: %w", err)
	}

	return cgroup, nil
}

func FromPath(path string) *CGroup {
	return &CGroup{
		path,
	}
}

func sanitize(subpath string) string {
	subpath = strings.ReplaceAll(subpath, "/", "")
	if subpath == ".." {
		return ""
	}
	return subpath
}
