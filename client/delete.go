package client

type DeleteOptions struct {
}

func (client *client) Delete(name string, opt *DeleteOptions) (err error) {
	entry, err := client.store.GetEntry(name)

	if err != nil {
		return
	}

	return entry.Delete()
}
