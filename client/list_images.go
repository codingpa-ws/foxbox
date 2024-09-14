package client

import (
	"os"
	"strings"
)

type Image struct {
	Name string
}

func (self *client) ListImages() (images []Image, err error) {
	dir := self.store.ImageBase()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && (strings.HasSuffix(name, ".tar") || strings.HasSuffix(name, ".tar.gz")) {
			name = strings.TrimSuffix(name, ".gz")
			name = strings.TrimSuffix(name, ".tar")

			images = append(images, Image{
				Name: name,
			})
		}
	}

	return
}
