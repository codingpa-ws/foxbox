package security

import (
	"fmt"
	"syscall"
)

func GetSysProcAttr(fd int, useCgroupFD bool) (*syscall.SysProcAttr, error) {
	uid, gid, err := GetUserIdentifiers()
	if err != nil {
		return nil, fmt.Errorf("getting uid/gid: %w", err)
	}

	return &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWPID | syscall.CLONE_NEWNET | syscall.CLONE_NEWTIME | syscall.CLONE_NEWCGROUP,
		Unshareflags: syscall.CLONE_NEWNS,
		UidMappings: []syscall.SysProcIDMap{{
			ContainerID: 0,
			HostID:      int(uid),
			Size:        1,
		}},
		GidMappings: []syscall.SysProcIDMap{{
			ContainerID: 0,
			HostID:      int(gid),
			Size:        1,
		}},
		CgroupFD:    fd,
		UseCgroupFD: useCgroupFD,
	}, nil
}
