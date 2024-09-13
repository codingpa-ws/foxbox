package client

import (
	"os"
	"slices"
)

type PsOptions struct {
	States []State
}

type ProcessInfo struct {
	ID       string
	PID      int
	State    State
	ExitCode int
}

func (client *client) Ps(opt *PsOptions) (infos []ProcessInfo, err error) {
	opt = newOr(opt)

	entries, err := os.ReadDir(client.store.EntryBase())
	if err != nil {
		return
	}
	for _, dir := range entries {
		if !dir.IsDir() {
			continue
		}

		entry, err := client.store.GetEntry(dir.Name())
		if err != nil {
			return nil, err
		}
		pid, running, err := entry.GetPID()
		if err != nil {
			return nil, err
		}

		var state State

		if running {
			state = StateRunning
		} else {
			state = StateStopped
		}

		if len(opt.States) > 0 && !slices.Contains(opt.States, state) {
			continue
		}

		infos = append(infos, ProcessInfo{
			ID:       dir.Name(),
			PID:      pid,
			State:    state,
			ExitCode: 0, // TODO: last exit code must be stored
		})
	}

	return
}
