package client_test

import (
	"testing"
	"time"

	"github.com/codingpa-ws/foxbox/client"
	"github.com/stretchr/testify/require"
)

func TestPs(t *testing.T) {
	require := require.New(t)
	store := newStore(t)
	downloadImage(t, store)

	foxbox := client.FromStore(store)
	name, err := foxbox.Create(&client.CreateOptions{
		Image: AlpineImageName,
	})
	require.NoError(err)

	runError := make(chan error)
	go func() {
		err = foxbox.Run(name, &client.RunOptions{
			Command: []string{"sleep", "0.05"},
		})
		runError <- err
		close(runError)
	}()

	time.Sleep(time.Millisecond * 10)

	infos, err := client.FromStore(store).Ps(nil)
	require.NoError(err)
	require.Equal(1, len(infos))
	info := infos[0]
	require.Equal(name, info.ID)
	require.Equal(client.StateRunning, info.State)
	require.Greater(info.PID, 0)

	require.NoError(<-runError)
}
