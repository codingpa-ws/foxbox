package testutil

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

type Fixture struct {
	name   string
	url    string
	sha256 string
}

func (self Fixture) path() string {
	const dir = "fixtures"
	_, base, _, _ := runtime.Caller(0)
	base = filepath.Dir(base)
	base = filepath.Dir(base)
	base = filepath.Dir(base)
	return filepath.Join(base, dir, self.name)
}

func (self Fixture) verifyHash() (ok bool, err error) {
	f, err := os.Open(self.path())
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer f.Close()
	hash := sha256.New()

	_, err = io.Copy(hash, f)
	if err != nil {
		return false, fmt.Errorf("reading file for hash verification: %s\n", err)
	}
	hexHash := fmt.Sprintf("%x", hash.Sum(nil))

	if hexHash != self.sha256 {
		return false, nil
	}

	return true, nil
}

func (self Fixture) downloadIfNeeded() error {
	ok, err := self.verifyHash()
	if err != nil {
		return fmt.Errorf("verifying hash: %w", err)
	}
	if ok {
		return nil
	}
	fmt.Printf("[foxbox test] Downloading fixture %s\n", self.name)

	osFile, err := os.OpenFile(self.path(), os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer osFile.Close()

	resp, err := http.Get(self.url)
	if err != nil {
		return fmt.Errorf("requesting %s: %w", self.url, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected status OK but got %d %s", resp.StatusCode, resp.Status)
	}

	defer resp.Body.Close()

	_, err = io.Copy(osFile, resp.Body)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", self.name, err)
	}

	ok, err = self.verifyHash()
	if err != nil {
		return fmt.Errorf("verifying hash after download: %w", err)
	}
	if !ok {
		return fmt.Errorf("mismatching hash for %s after download", self.name)
	}

	return nil
}

func (self Fixture) get() (io.ReadCloser, error) {
	err := self.downloadIfNeeded()
	if err != nil {
		return nil, err
	}

	return os.Open(self.path())
}

func DownloadAlpineImage() (io.ReadCloser, error) {
	return Fixture{
		name:   "alpine-minirootfs-3.18.4-x86_64.tar.gz",
		url:    "https://dl-cdn.alpinelinux.org/alpine/v3.18/releases/x86_64/alpine-minirootfs-3.18.4-x86_64.tar.gz",
		sha256: "c59d5203bc6b8b6ef81f3f6b63e32c28d6e47be806ba8528f8766a4ca506c7ba",
	}.get()
}
