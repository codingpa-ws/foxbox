package client

import (
	"os"
	"path/filepath"

	"github.com/codingpa-ws/foxbox/internal/store"
)

const RuntimeDir = "runtime"

type Client interface {
	Create(opt *CreateOptions) (name string, err error)
	Delete(name string, opt *DeleteOptions) (err error)
	List(opt *ListOptions) (ids []string, err error)
	Ps(opt *PsOptions) (infos []ProcessInfo, err error)
	Run(name string, opt *RunOptions) (err error)
}

type client struct {
	store *store.Store
}

func FromStore(store *store.Store) Client {
	return &client{store}
}

func FromUser() (Client, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, ".local", "share", "containers", "foxbox")

	err = os.MkdirAll(path, 0755)
	if err != nil {
		return nil, err
	}
	store, err := store.New(path)
	if err != nil {
		return nil, err
	}

	return &client{store}, nil
}

type State string

const (
	StateRunning State = "running"
	StateStopped State = "stopped"
	StateExited  State = "exited"
)

// Returns the given value of type *T or,
// if it is nil, returns a new(T).
func newOr[T any](t *T) *T {
	if t == nil {
		return new(T)
	}

	return t
}
