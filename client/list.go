package client

import (
	"os"

	"github.com/codingpa-ws/foxbox/internal/store"
)

type ListOptions struct {
	Store *store.Store
}

func List(opt *ListOptions) (ids []string, err error) {
	opt = newOr(opt)

	if opt.Store == nil {
		opt.Store, err = store.New(RuntimeDir)
	}

	path := opt.Store.EntryBase()

	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			ids = append(ids, entry.Name())
		}
	}

	return
}
