package client_test

import (
	"testing"
	"time"

	"github.com/codingpa-ws/foxbox/client"
	"github.com/stretchr/testify/require"
)

func TestPs(t *testing.T) {
	require := require.New(t)
	store, deleteStore := downloadImage(t)
	defer deleteStore()

	name, err := client.Create(&client.CreateOptions{
		Image: AlpineImageName,
		Store: store,
	})
	require.NoError(err)

	runError := make(chan error)
	go func() {
		err = client.Run(name, &client.RunOptions{
			Store:   store,
			Command: []string{"sleep", "0.05"},
		})
		runError <- err
		close(runError)
	}()

	time.Sleep(time.Millisecond * 10)

	infos, err := client.Ps(&client.PsOptions{
		Store: store,
	})
	require.NoError(err)
	require.Equal(1, len(infos))
	info := infos[0]
	require.Equal(name, info.ID)
	require.Equal(client.StateRunning, info.State)
	require.Greater(info.PID, 0)

	require.NoError(<-runError)
}
