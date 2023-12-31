package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Store struct{ base string }

func New(base string) (*Store, error) {
	store := Store{base}

	return &store, store.init()
}

func (self Store) Base() string {
	return self.base
}

func (self Store) EntryBase() string {
	return filepath.Join(self.base, "entries")
}

func (self Store) ImageBase() string {
	return filepath.Join(self.base, "images")
}

func (self Store) GetEntry(name string) (*StoreEntry, error) {
	name = sanitize(name)
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("store: invalid foxbox name")
	}
	entry := &StoreEntry{filepath.Join(self.EntryBase(), name)}

	_, err := os.Stat(entry.base)
	if os.IsNotExist(err) || err != nil {
		return nil, err
	}

	return entry, entry.init()
}

func (self Store) NewEntry(name string) (*StoreEntry, error) {
	entry := &StoreEntry{filepath.Join(self.EntryBase(), sanitize(name))}

	_, err := os.Stat(entry.base)
	if os.IsExist(err) || !os.IsNotExist(err) {
		return nil, err
	}

	return entry, entry.init()
}

func (self Store) GetImagePath(name string, gzip bool) string {
	path := filepath.Join(self.ImageBase(), sanitize(name)) + ".tar"
	if gzip {
		path = path + ".gz"
	}
	return path
}

type StoreEntry struct{ base string }

func (self Store) init() error {
	return errors.Join(
		os.MkdirAll(self.EntryBase(), 0755),
		os.MkdirAll(self.ImageBase(), 0755),
	)
}

func (self StoreEntry) Base() string {
	return self.base
}

func (self StoreEntry) FileSystem() string {
	return filepath.Join(self.base, "boxfs")
}

func (self StoreEntry) SetPID(pid int) error {
	str := strconv.Itoa(pid)
	path := filepath.Join(self.Base(), "container.pid")
	return os.WriteFile(path, []byte(str), 0644)
}

func (self StoreEntry) GetPID() (pid int, running bool, err error) {
	path := filepath.Join(self.Base(), "container.pid")
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, false, err
	}
	pid, err = strconv.Atoi(string(b))
	if err != nil {
		return 0, false, err
	}
	_, err = os.Stat(fmt.Sprintf("/proc/%d", pid))
	if os.IsNotExist(err) {
		return pid, false, nil
	}
	if err != nil {
		return
	}

	return pid, true, nil
}

func (self StoreEntry) init() error {
	return os.MkdirAll(self.FileSystem(), 0755)
}

func (self StoreEntry) Delete() error {
	return os.RemoveAll(self.base)
}

func sanitize(subpath string) string {
	return strings.ReplaceAll(subpath, "/", "")
}
