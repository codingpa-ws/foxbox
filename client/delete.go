package client

import "github.com/codingpa-ws/foxbox/internal/store"

type DeleteOptions struct {
	Store *store.Store
}

func Delete(name string, opt *DeleteOptions) (err error) {
	opt = newOr(opt)

	if opt.Store == nil {
		opt.Store, err = store.New("runtime")
		if err != nil {
			return
		}
	}

	entry, err := opt.Store.GetEntry(name)

	if err != nil {
		return
	}

	return entry.Delete()
}
