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

	store := newStore(t)
	downloadImage(t, store)

	foxbox := client.FromStore(store)
	name, err := foxbox.Create(&client.CreateOptions{
		Image: AlpineImageName,
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
		{
			command: []string{"ls", "/opt/foxbox"},
			stdout:  "main.go\n",
		},
	}

	for _, command := range commands {
		t.Run("command "+strings.Join(command.command, " "), func(t *testing.T) {
			stdout, stderr, err := run(foxbox, name, command.command...)

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

	err = foxbox.Delete(name, nil)
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

		store := newStore(t)
		downloadImage(t, store)

		foxbox := client.FromStore(store)
		name, err := foxbox.Create(&client.CreateOptions{
			Image: AlpineImageName,
		})
		require.NoError(err)

		firstRunErr := make(chan error)
		go func() {
			fmt.Println("start")
			err := foxbox.Run(name, &client.RunOptions{
				Command: []string{"sleep", "0.1"},
			})
			fmt.Println("end")
			firstRunErr <- err
		}()
		// yeah, currently vulnerable to race condition
		time.Sleep(time.Millisecond * 10)

		err = foxbox.Run(name, &client.RunOptions{
			Command: []string{"ls"},
		})
		require.Error(err, "client must prevent running a container twice in parallel")
		require.NoError(<-firstRunErr)
	})
}

func run(
	foxbox client.Client,
	name string,
	command ...string,
) (stdout, stderr string, err error) {
	stdoutBuilder := new(strings.Builder)
	stderrBuilder := new(strings.Builder)
	err = foxbox.Run(name, &client.RunOptions{
		Stdout:           stdoutBuilder,
		Stderr:           stderrBuilder,
		Command:          command,
		EnableNetworking: true,
		Volumes: []client.VolumeConfig{{
			HostPath: "../cmd/foxbox",
			BoxPath:  "/opt/foxbox",
		}},
	})
	if err != nil {
		fmt.Println("stdout", stdoutBuilder.String())
		fmt.Println("stderr", stderrBuilder.String())
	}
	return stdoutBuilder.String(), stderrBuilder.String(), err
}

func newStore(t *testing.T) *store.Store {
	require := require.New(t)
	base, err := os.MkdirTemp("", "")
	require.NoError(err)
	store, err := store.New(base)
	require.NoError(err)
	t.Cleanup(func() {
		os.RemoveAll(base)
	})
	return store
}

func downloadImage(t *testing.T, store *store.Store) {
	require := require.New(t)
	alpineImage, err := testutil.DownloadAlpineImage()
	require.NoError(err)
	defer alpineImage.Close()

	file, err := os.OpenFile(store.GetImagePath(AlpineImageName, true), os.O_CREATE|os.O_WRONLY, 0644)
	require.NoError(err)
	defer file.Close()

	_, err = io.Copy(file, alpineImage)
	require.NoError(err)
}
