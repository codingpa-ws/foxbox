package client

import (
	"os"
	"slices"

	"github.com/codingpa-ws/foxbox/internal/store"
)

type PsOptions struct {
	States []State
	Store  *store.Store
}

type ProcessInfo struct {
	ID       string
	PID      int
	State    State
	ExitCode int
}

func Ps(opt *PsOptions) (infos []ProcessInfo, err error) {
	opt = newOr(opt)

	if opt.Store == nil {
		opt.Store, err = store.New("runtime")
		if err != nil {
			return
		}
	}

	entries, err := os.ReadDir(opt.Store.EntryBase())
	if err != nil {
		return
	}
	for _, dir := range entries {
		if !dir.IsDir() {
			continue
		}

		entry, err := opt.Store.GetEntry(dir.Name())
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
