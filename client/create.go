package client

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"

	"github.com/codingpa-ws/foxbox/internal/store"
	"github.com/klauspost/pgzip"
)

type CreateOptions struct {
	Image string
	Store *store.Store
}

func (self *CreateOptions) GetImage() (f io.ReadCloser, gzipped bool, err error) {
	path := self.Store.GetImagePath(self.Image, false)

	f, err = os.Open(path)
	if err != nil {
		f, err = os.Open(path + ".gz")
		gzipped = true
	}

	return
}

func Create(opt *CreateOptions) (name string, err error) {
	opt = newOr(opt)
	name = NewName()

	if opt.Store == nil {
		opt.Store, err = store.New("runtime")
		if err != nil {
			return
		}
	}

	entry, err := opt.Store.NewEntry(name)
	if err != nil {
		return
	}

	image, gzipped, err := opt.GetImage()
	if err != nil {
		entry.Delete()
		return
	}

	err = extractImage(image, gzipped, entry.FileSystem())

	if err != nil {
		entry.Delete()
		return
	}

	err = setupResolvConf(entry)
	if err != nil {
		entry.Delete()
		return
	}

	return
}

func extractImage(image io.ReadCloser, ungzip bool, path string) error {
	if ungzip {
		var err error
		image, err = pgzip.NewReader(image)
		if err != nil {
			return err
		}
	}
	defer image.Close()

	tr := tar.NewReader(image)

	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}

		fpath := filepath.Join(path, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(fpath); err != nil {
				if err := os.MkdirAll(fpath, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			f, err := os.OpenFile(fpath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			defer f.Close()

			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.Symlink(header.Linkname, fpath); err != nil {
				return err
			}
		}
	}
}

const resolvConf = "nameserver 10.0.2.3\n"

func setupResolvConf(entry *store.StoreEntry) error {
	path := filepath.Join(entry.FileSystem(), "etc", "resolv.conf")
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		return nil
	}
	return os.WriteFile(path, []byte(resolvConf), 0644)
}
