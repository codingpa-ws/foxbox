package security

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// Uses prctl(PR_CAPBSET_DROP) to drop capabilities from the process’
// capability bounding set.
//
// See capabilities(7) for a list of all available capabilities
func DropCapabilities() error {
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
