package store_test

import (
	"os"
	"strings"
	"testing"

	"github.com/codingpa-ws/foxbox/internal/store"
	"github.com/stretchr/testify/require"
)

func mustStore(t *testing.T) (*store.Store, func()) {
	dir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	store, err := store.New(dir)
	if err != nil {
		require.NoError(t, err)
	}

	return store, func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)
	}
}

func scandir(t *testing.T, path string) (result []string) {
	entries, err := os.ReadDir(path)
	require.NoError(t, err)
	for _, entry := range entries {
		result = append(result, entry.Name())
	}
	return
}

func assertDirContents(t *testing.T, dir string, expected []string) {
	actual := scandir(t, dir)
	if len(expected) == 0 {
		require.Empty(t, actual)
	} else {
		require.EqualValues(t, expected, actual)
	}
}

func TestNew(t *testing.T) {
	store, removeStore := mustStore(t)
	defer removeStore()

	assertDirContents(t, store.Base(), []string{"entries", "images"})
	assertDirContents(t, store.EntryBase(), []string{})

	require.Truef(t, strings.HasSuffix(store.EntryBase(), "/entries"), "wanted suffix /entries, got %s", store.EntryBase())

	entry, err := store.NewEntry("testbox")
	require.NoError(t, err)
	require.Equal(t, store.EntryBase()+"/testbox", entry.Base())

	assertDirContents(t, store.EntryBase(), []string{"testbox"})

	require.Equal(t, entry.Base()+"/boxfs", entry.FileSystem())

	err = entry.Delete()
	require.NoError(t, err)

	assertDirContents(t, store.EntryBase(), []string{})
}
