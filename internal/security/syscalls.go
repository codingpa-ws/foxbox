package security

import (
	"fmt"
	"syscall"

	seccomp "github.com/seccomp/libseccomp-golang"
	"golang.org/x/sys/unix"
)

var forbiddenSyscalls = []string{
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

func RestrictSyscalls() error {
	actionFail := seccomp.ActErrno.SetReturnCode(int16(unix.EPERM))
	filter, err := seccomp.NewFilter(seccomp.ActAllow)
	if err != nil {
		return fmt.Errorf("creating seccomp filter: %w", err)
	}
	defer filter.Release()

	rules := []struct {
		syscall   string
		action    seccomp.ScmpAction
		condition seccomp.ScmpCondition
	}{
		{
			syscall: "chmod",
			action:  actionFail,
			condition: seccomp.ScmpCondition{
				Argument: 1,
				Operand1: syscall.S_ISUID,
				Operand2: syscall.S_ISUID,
				Op:       seccomp.CompareMaskedEqual,
			},
		},
		{
			syscall: "chmod",
			action:  actionFail,
			condition: seccomp.ScmpCondition{
				Argument: 1,
				Operand1: syscall.S_ISGID,
				Operand2: syscall.S_ISGID,
				Op:       seccomp.CompareMaskedEqual,
			},
		},
		{
			syscall: "fchmod",
			action:  actionFail,
			condition: seccomp.ScmpCondition{
				Argument: 1,
				Operand1: syscall.S_ISUID,
				Operand2: syscall.S_ISUID,
				Op:       seccomp.CompareMaskedEqual,
			},
		},
		{
			syscall: "fchmod",
			action:  actionFail,
			condition: seccomp.ScmpCondition{
				Argument: 1,
				Operand1: syscall.S_ISGID,
				Operand2: syscall.S_ISGID,
				Op:       seccomp.CompareMaskedEqual,
			},
		},
		{
			syscall: "fchmodat",
			action:  actionFail,
			condition: seccomp.ScmpCondition{
				Argument: 2,
				Operand1: syscall.S_ISUID,
				Operand2: syscall.S_ISUID,
				Op:       seccomp.CompareMaskedEqual,
			},
		},
		{
			syscall: "fchmodat",
			action:  actionFail,
			condition: seccomp.ScmpCondition{
				Argument: 2,
				Operand1: syscall.S_ISGID,
				Operand2: syscall.S_ISGID,
				Op:       seccomp.CompareMaskedEqual,
			},
		},
		{
			syscall: "unshare",
			action:  actionFail,
			condition: seccomp.ScmpCondition{
				Argument: 0,
				Operand1: syscall.CLONE_NEWUSER,
				Operand2: syscall.CLONE_NEWUSER,
				Op:       seccomp.CompareMaskedEqual,
			},
		},
		{
			syscall: "clone",
			action:  actionFail,
			condition: seccomp.ScmpCondition{
				Argument: 0,
				Operand1: syscall.CLONE_NEWUSER,
				Operand2: syscall.CLONE_NEWUSER,
				Op:       seccomp.CompareMaskedEqual,
			},
		},
		{
			syscall: "ioctl",
			action:  actionFail,
			condition: seccomp.ScmpCondition{
				Argument: 0,
				Operand1: syscall.CLONE_NEWUSER,
				Operand2: syscall.CLONE_NEWUSER,
				Op:       seccomp.CompareMaskedEqual,
			},
		},
	}

	for _, rule := range rules {
		call, err := seccomp.GetSyscallFromName(rule.syscall)
		if err != nil {
			return fmt.Errorf("getting syscall `%s`: %w", rule.syscall, err)
		}

		err = filter.AddRuleConditional(call, rule.action, []seccomp.ScmpCondition{
			rule.condition,
		})
		if err != nil {
			return fmt.Errorf("adding rule for %s: %w", rule.syscall, err)
		}
	}

	err = filter.SetNoNewPrivsBit(false)
	if err != nil {
		return fmt.Errorf("setting SCMP_FLTATR_CTL_NNP: %w", err)
	}

	for _, name := range forbiddenSyscalls {
		call, err := seccomp.GetSyscallFromName(name)

		if err != nil {
			return fmt.Errorf("getting syscall `%s`: %w", name, err)
		}
		err = filter.AddRule(call, actionFail)

		if err != nil {
			return fmt.Errorf("adding rule for `%s`: %w", name, err)
		}
	}

	return filter.Load()
}
