package slirp

import (
	"fmt"
	"os/exec"
	"strconv"
)

func Start(pid int) (*exec.Cmd, error) {
	cmd := exec.Command("slirp4netns", "--configure", "--mtu=65520", "--disable-host-loopback", strconv.Itoa(pid), "top0")

	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("starting slirp: %w", err)
	}

	return cmd, nil
}
