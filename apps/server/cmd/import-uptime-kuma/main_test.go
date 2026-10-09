package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestOpenSQLiteModes(t *testing.T) {
	sourcePath := filepath.Join(".", ".uptime-kuma-source-"+uuid.NewString()+".db")
	targetPath := filepath.Join(".", ".uptime-kuma-target-"+uuid.NewString()+".db")
	t.Cleanup(func() {
		require.NoError(t, os.Remove(sourcePath))
		require.NoError(t, os.Remove(targetPath))
	})
	require.NoError(t, os.WriteFile(sourcePath, nil, 0600))
	require.NoError(t, os.WriteFile(targetPath, nil, 0600))

	source, err := openSQLite(sourcePath, true)
	require.NoError(t, err)
	defer source.Close()
	_, err = source.Exec("CREATE TABLE should_not_write (id INTEGER)")
	require.Error(t, err)

	target, err := openSQLite(targetPath, false)
	require.NoError(t, err)
	defer target.Close()
	_, err = target.Exec("CREATE TABLE should_write (id INTEGER)")
	require.NoError(t, err)
}
