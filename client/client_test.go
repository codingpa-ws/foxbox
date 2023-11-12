package client_test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/codingpa-ws/foxbox/client"
	"github.com/codingpa-ws/foxbox/internal/store"
	"github.com/stretchr/testify/require"
)

const AlpineImageURL = "https://dl-cdn.alpinelinux.org/alpine/v3.18/releases/x86_64/alpine-minirootfs-3.18.4-x86_64.tar.gz"
const AlpineImageName = "alpine-3.18.4-x86_64"

func TestIntegration(t *testing.T) {
	require := require.New(t)
	if testing.Short() {
		t.Skip("integration test is slow")
	}

	store, deleteStore := downloadImage(t)
	defer deleteStore()

	name, err := client.Create(&client.CreateOptions{
		Image: AlpineImageName,
		Store: store,
	})
	require.NoError(err)
	require.NotEmpty(name)

	entry, err := store.GetEntry(name)
	require.NoError(err)

	stdout := new(strings.Builder)
	stderr := new(strings.Builder)

	err = client.Run(name, &client.RunOptions{
		Stdout:  stdout,
		Stderr:  stderr,
		Store:   store,
		Command: []string{"ls"},
	})
	if err != nil {
		fmt.Println("stdout", stdout.String())
		fmt.Println("stderr", stderr.String())
	}
	require.NoError(err)
	require.Equal("bin\ndev\netc\nhome\nlib\nmedia\nmnt\nopt\nproc\nroot\nrun\nsbin\nsrv\nsys\ntmp\nusr\nvar\n", stdout.String())
	require.Equal("", stderr.String())

	stdout = new(strings.Builder)
	stderr = new(strings.Builder)

	err = client.Run(name, &client.RunOptions{
		Stdout:  stdout,
		Stderr:  stderr,
		Store:   store,
		Command: []string{"ps", "aux"},
	})
	if err != nil {
		fmt.Println("stdout", stdout.String())
		fmt.Println("stderr", stderr.String())
	}
	require.NoError(err)
	require.Equal("PID   USER     TIME  COMMAND\n    1 root      0:00 {sh} ps aux\n", stdout.String())
	require.Equal("", stderr.String())

	stdout = new(strings.Builder)
	stderr = new(strings.Builder)

	err = client.Run(name, &client.RunOptions{
		Stdout:  stdout,
		Stderr:  stderr,
		Store:   store,
		Command: []string{"sh", "-c", `echo "name=$(whoami),uid=$(id -u),gid=$(id -g)"`},
	})
	require.NoError(err)
	require.Equal("name=root,uid=0,gid=0\n", stdout.String())
	require.Equal("", stderr.String())

	info, err := os.Stat(entry.FileSystem())
	require.NoError(err)
	require.Truef(info.IsDir(), "%s (box filesystem path) must be a directory", entry.FileSystem())

	err = client.Delete(name, &client.DeleteOptions{
		Store: store,
	})
	require.NoError(err)

	_, err = os.Stat(entry.FileSystem())
	require.ErrorIs(err, os.ErrNotExist, "client.Delete(string) didn’t delete container")
}

func downloadImage(t *testing.T) (*store.Store, func()) {
	require := require.New(t)
	base, err := os.MkdirTemp("", "")
	require.NoError(err)
	store, err := store.New(base)
	require.NoError(err)

	resp, err := http.Get(AlpineImageURL)
	require.NoError(err)
	require.Equal(http.StatusOK, resp.StatusCode)

	defer resp.Body.Close()

	file, err := os.OpenFile(store.GetImagePath(AlpineImageName, true), os.O_CREATE|os.O_WRONLY, 0644)
	require.NoError(err)
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	require.NoError(err)

	return store, func() {
		require.NoError(os.RemoveAll(base))
	}
}
