package client

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

	"github.com/codingpa-ws/foxbox/internal/store"
)

type CreateOptions struct {
	Image string
	Store *store.Store
}

func (self *CreateOptions) GetImage() (f io.ReadCloser, gzipped bool, err error) {
	path := filepath.Join("images", self.Image+".tar")

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
	}

	return
}

func extractImage(image io.ReadCloser, ungzip bool, path string) error {
	if ungzip {
		var err error
		image, err = gzip.NewReader(image)
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

			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			f.Close()
		case tar.TypeSymlink:
			if err := os.Symlink(header.Linkname, fpath); err != nil {
				return err
			}
		}
	}
}
