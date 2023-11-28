package client_test

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/codingpa-ws/foxbox/client"
	"github.com/codingpa-ws/foxbox/internal/store"
	"github.com/codingpa-ws/foxbox/internal/testutil"
	"github.com/stretchr/testify/require"
)

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
		err     string
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
		{
			command: []string{"hostname"},
			stdout:  name + "\n",
		},
		{
			command: []string{"env"},
			stdout:  "PATH=/bin:/sbin:/usr/bin:/usr/sbin\nLANG=C.UTF-8\nCHARSET=UTF-8\n",
		},
		{
			command: []string{"sh", "-c", `apk update > /dev/null; apk add tree -s`},
			stdout:  "(1/1) Installing tree (2.1.1-r0)\nOK: 7 MiB in 15 packages\n",
		},
		{
			command: []string{"sh", "-c", "exit 42"},
			err:     "exit status 42",
		},
	}

	for _, command := range commands {
		t.Run("command "+strings.Join(command.command, " "), func(t *testing.T) {
			stdout, stderr, err := run(t, name, store, command.command...)

			var errString string
			if err != nil {
				errString = err.Error()
			}
			require.Equal(t, command.err, errString)
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

func TestRun(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test is slow")
	}
	t.Run("in parallel must fail", func(t *testing.T) {
		require := require.New(t)

		store, deleteStore := downloadImage(t)
		defer deleteStore()

		name, err := client.Create(&client.CreateOptions{
			Image: AlpineImageName,
			Store: store,
		})
		require.NoError(err)

		firstRunErr := make(chan error)
		go func() {
			fmt.Println("start")
			err := client.Run(name, &client.RunOptions{
				Command: []string{"sleep", "0.1"},
				Store:   store,
			})
			fmt.Println("end")
			firstRunErr <- err
		}()
		// yeah, currently vulnerable to race condition
		time.Sleep(time.Millisecond * 10)

		err = client.Run(name, &client.RunOptions{
			Command: []string{"ls"},
			Store:   store,
		})
		require.Error(err, "client must prevent running a container twice in parallel")
		require.NoError(<-firstRunErr)
	})
}

func run(
	t *testing.T,
	name string,
	store *store.Store,
	command ...string,
) (stdout, stderr string, err error) {
	stdoutBuilder := new(strings.Builder)
	stderrBuilder := new(strings.Builder)
	err = client.Run(name, &client.RunOptions{
		Stdout:           stdoutBuilder,
		Stderr:           stderrBuilder,
		Store:            store,
		Command:          command,
		EnableNetworking: true,
	})
	if err != nil {
		fmt.Println("stdout", stdoutBuilder.String())
		fmt.Println("stderr", stderrBuilder.String())
	}
	return stdoutBuilder.String(), stderrBuilder.String(), err
}

func downloadImage(t *testing.T) (*store.Store, func()) {
	require := require.New(t)
	base, err := os.MkdirTemp("", "")
	require.NoError(err)
	store, err := store.New(base)
	require.NoError(err)

	alpineImage, err := testutil.DownloadAlpineImage()
	require.NoError(err)
	defer alpineImage.Close()

	file, err := os.OpenFile(store.GetImagePath(AlpineImageName, true), os.O_CREATE|os.O_WRONLY, 0644)
	require.NoError(err)
	defer file.Close()

	_, err = io.Copy(file, alpineImage)
	require.NoError(err)

	return store, func() {
		require.NoError(os.RemoveAll(base))
	}
}
