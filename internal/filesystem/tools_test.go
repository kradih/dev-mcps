package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/local-mcps/dev-mcps/config"
)

func newTestServer(t *testing.T, tempDir string) *Server {
	cfg := &config.FilesystemConfig{
		AllowedPaths:   []string{tempDir},
		DeniedPaths:    []string{},
		MaxFileSizeMB:  10,
		FollowSymlinks: true,
	}
	return NewServer(cfg)
}

func TestReadFile(t *testing.T) {
	tempDir := t.TempDir()
	server := newTestServer(t, tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := "Hello, World!"
	require.NoError(t, os.WriteFile(testFile, []byte(content), 0644))

	t.Run("read existing file", func(t *testing.T) {
		result, err := server.handleReadFile(context.Background(), map[string]interface{}{
			"path": testFile,
		})
		require.NoError(t, err)
		assert.Equal(t, content, result.Content[0].Text)
	})

	t.Run("read nonexistent file", func(t *testing.T) {
		_, err := server.handleReadFile(context.Background(), map[string]interface{}{
			"path": filepath.Join(tempDir, "nonexistent.txt"),
		})
		assert.Error(t, err)
	})

	t.Run("missing path parameter", func(t *testing.T) {
		_, err := server.handleReadFile(context.Background(), map[string]interface{}{})
		assert.Error(t, err)
	})
}

func TestWriteFile(t *testing.T) {
	tempDir := t.TempDir()
	server := newTestServer(t, tempDir)

	t.Run("write new file", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "new.txt")
		content := "New content"

		result, err := server.handleWriteFile(context.Background(), map[string]interface{}{
			"path":    testFile,
			"content": content,
		})
		require.NoError(t, err)
		assert.Contains(t, result.Content[0].Text, "Successfully wrote")

		data, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("overwrite existing file", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "existing.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("old"), 0644))

		newContent := "new content"
		_, err := server.handleWriteFile(context.Background(), map[string]interface{}{
			"path":    testFile,
			"content": newContent,
		})
		require.NoError(t, err)

		data, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, newContent, string(data))
	})

	t.Run("create parent directories", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "subdir", "deep", "file.txt")
		content := "nested content"

		_, err := server.handleWriteFile(context.Background(), map[string]interface{}{
			"path":    testFile,
			"content": content,
		})
		require.NoError(t, err)

		data, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})
}

func TestListDirectory(t *testing.T) {
	tempDir := t.TempDir()
	server := newTestServer(t, tempDir)

	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("1"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("2"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, ".hidden"), []byte("h"), 0644))
	require.NoError(t, os.Mkdir(filepath.Join(tempDir, "subdir"), 0755))

	t.Run("list directory without hidden", func(t *testing.T) {
		result, err := server.handleListDirectory(context.Background(), map[string]interface{}{
			"path":           tempDir,
			"include_hidden": false,
		})
		require.NoError(t, err)
		assert.Contains(t, result.Content[0].Text, "file1.txt")
		assert.Contains(t, result.Content[0].Text, "file2.txt")
		assert.Contains(t, result.Content[0].Text, "subdir")
		assert.NotContains(t, result.Content[0].Text, ".hidden")
	})

	t.Run("list directory with hidden", func(t *testing.T) {
		result, err := server.handleListDirectory(context.Background(), map[string]interface{}{
			"path":           tempDir,
			"include_hidden": true,
		})
		require.NoError(t, err)
		assert.Contains(t, result.Content[0].Text, ".hidden")
	})
}

func TestCreateDirectory(t *testing.T) {
	tempDir := t.TempDir()
	server := newTestServer(t, tempDir)

	t.Run("create new directory", func(t *testing.T) {
		newDir := filepath.Join(tempDir, "newdir")
		_, err := server.handleCreateDirectory(context.Background(), map[string]interface{}{
			"path": newDir,
		})
		require.NoError(t, err)

		info, err := os.Stat(newDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("create nested directories", func(t *testing.T) {
		nestedDir := filepath.Join(tempDir, "a", "b", "c")
		_, err := server.handleCreateDirectory(context.Background(), map[string]interface{}{
			"path": nestedDir,
		})
		require.NoError(t, err)

		info, err := os.Stat(nestedDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}

func TestDeleteFile(t *testing.T) {
	tempDir := t.TempDir()
	server := newTestServer(t, tempDir)

	t.Run("delete existing file", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "todelete.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("delete me"), 0644))

		_, err := server.handleDeleteFile(context.Background(), map[string]interface{}{
			"path": testFile,
		})
		require.NoError(t, err)

		_, err = os.Stat(testFile)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("delete nonexistent file", func(t *testing.T) {
		_, err := server.handleDeleteFile(context.Background(), map[string]interface{}{
			"path": filepath.Join(tempDir, "nonexistent.txt"),
		})
		assert.Error(t, err)
	})
}

func TestFileInfo(t *testing.T) {
	tempDir := t.TempDir()
	server := newTestServer(t, tempDir)

	testFile := filepath.Join(tempDir, "info.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0644))

	result, err := server.handleFileInfo(context.Background(), map[string]interface{}{
		"path": testFile,
	})
	require.NoError(t, err)
	assert.Contains(t, result.Content[0].Text, "info.txt")
	assert.Contains(t, result.Content[0].Text, "size_bytes")
}

func TestCopyFile(t *testing.T) {
	tempDir := t.TempDir()
	server := newTestServer(t, tempDir)

	srcFile := filepath.Join(tempDir, "source.txt")
	dstFile := filepath.Join(tempDir, "destination.txt")
	content := "copy this content"

	require.NoError(t, os.WriteFile(srcFile, []byte(content), 0644))

	_, err := server.handleCopyFile(context.Background(), map[string]interface{}{
		"source":      srcFile,
		"destination": dstFile,
	})
	require.NoError(t, err)

	data, err := os.ReadFile(dstFile)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))

	_, err = os.Stat(srcFile)
	assert.NoError(t, err)
}

func TestMoveFile(t *testing.T) {
	tempDir := t.TempDir()
	server := newTestServer(t, tempDir)

	srcFile := filepath.Join(tempDir, "tomove.txt")
	dstFile := filepath.Join(tempDir, "moved.txt")
	content := "move this content"

	require.NoError(t, os.WriteFile(srcFile, []byte(content), 0644))

	_, err := server.handleMoveFile(context.Background(), map[string]interface{}{
		"source":      srcFile,
		"destination": dstFile,
	})
	require.NoError(t, err)

	data, err := os.ReadFile(dstFile)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))

	_, err = os.Stat(srcFile)
	assert.True(t, os.IsNotExist(err))
}
