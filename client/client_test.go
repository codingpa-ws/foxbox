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
	requires := require.New(t)
	if testing.Short() {
		t.Skip("integration test is slow")
	}

	store, deleteStore := downloadImage(t)
	defer deleteStore()

	name, err := client.Create(&client.CreateOptions{
		Image: AlpineImageName,
		Store: store,
	})
	requires.NoError(err)
	requires.NotEmpty(name)

	entry, err := store.GetEntry(name)
	requires.NoError(err)

	commands := []struct {
		command []string
		stdout  string
		stderr  string
	}{
		{
			command: []string{"ls"},
			stdout:  "bin\ndev\netc\nhome\nlib\nmedia\nmnt\nopt\nproc\nroot\nrun\nsbin\nsrv\nsys\ntmp\nusr\nvar\n",
		},
		{
			command: []string{"ps", "aux"},
			stdout:  "PID   USER     TIME  COMMAND\n    1 root      0:00 {sh} ps aux\n",
		},
		{
			command: []string{"sh", "-c", `echo "name=$(whoami),uid=$(id -u),gid=$(id -g)"`},
			stdout:  "name=root,uid=0,gid=0\n",
		},
		{
			command: []string{"pwd"},
			stdout:  "/\n",
		},
	}

	for _, command := range commands {
		t.Run("command "+strings.Join(command.command, " "), func(t *testing.T) {
			stdout, stderr := run(t, name, store, command.command...)

			require.Equal(t, command.stdout, stdout)
			require.Equal(t, command.stderr, stderr)
		})
	}

	info, err := os.Stat(entry.FileSystem())
	requires.NoError(err)
	requires.Truef(info.IsDir(), "%s (box filesystem path) must be a directory", entry.FileSystem())

	err = client.Delete(name, &client.DeleteOptions{
		Store: store,
	})
	requires.NoError(err)

	_, err = os.Stat(entry.FileSystem())
	requires.ErrorIs(err, os.ErrNotExist, "client.Delete(string) didn’t delete container")
}

func run(
	t *testing.T,
	name string,
	store *store.Store,
	command ...string,
) (stdout, stderr string) {
	stdoutBuilder := new(strings.Builder)
	stderrBuilder := new(strings.Builder)
	err := client.Run(name, &client.RunOptions{
		Stdout:  stdoutBuilder,
		Stderr:  stderrBuilder,
		Store:   store,
		Command: command,
	})
	if err != nil {
		fmt.Println("stdout", stdoutBuilder.String())
		fmt.Println("stderr", stderrBuilder.String())
	}
	require.NoError(t, err)
	return stdoutBuilder.String(), stderrBuilder.String()
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
