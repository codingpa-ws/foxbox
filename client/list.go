package client

import (
	"os"
)

type ListOptions struct {
}

func (client *client) List(opt *ListOptions) (ids []string, err error) {
	path := client.store.EntryBase()

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
