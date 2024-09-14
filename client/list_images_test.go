package client_test

import (
	"testing"

	"github.com/codingpa-ws/foxbox/client"
	"github.com/stretchr/testify/require"
)

func TestListImages(t *testing.T) {
	require := require.New(t)
	store := newStore(t)

	foxbox := client.FromStore(store)
	images, err := foxbox.ListImages()
	require.NoError(err)
	require.Empty(images)

	downloadImage(t, store)

	images, err = foxbox.ListImages()
	require.NoError(err)
	require.Equal([]client.Image{
		{
			Name: AlpineImageName,
		},
	}, images)
}
