package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathValidator(t *testing.T) {
	homeDir, _ := os.UserHomeDir()

	t.Run("validate allowed path", func(t *testing.T) {
		v := NewPathValidator([]string{homeDir}, nil, true)
		err := v.ValidatePath(homeDir)
		assert.NoError(t, err)
	})

	t.Run("validate denied path", func(t *testing.T) {
		sshDir := filepath.Join(homeDir, ".ssh")
		v := NewPathValidator([]string{homeDir}, []string{sshDir}, true)
		err := v.ValidatePath(sshDir)
		assert.Error(t, err)
		assert.True(t, IsPathNotAllowed(err))
	})

	t.Run("validate path not in allowed list", func(t *testing.T) {
		v := NewPathValidator([]string{"/tmp"}, nil, true)
		err := v.ValidatePath(homeDir)
		assert.Error(t, err)
	})

	t.Run("empty path", func(t *testing.T) {
		v := NewPathValidator([]string{homeDir}, nil, true)
		err := v.ValidatePath("")
		assert.Error(t, err)
	})

	t.Run("empty allowed list allows all", func(t *testing.T) {
		v := NewPathValidator(nil, nil, true)
		err := v.ValidatePath("/tmp")
		assert.NoError(t, err)
	})
}

func TestCommandValidator(t *testing.T) {
	t.Run("allow any command with empty lists", func(t *testing.T) {
		v := NewCommandValidator(nil, nil)
		err := v.ValidateCommand("ls", []string{"-la"})
		assert.NoError(t, err)
	})

	t.Run("deny dangerous command", func(t *testing.T) {
		v := NewCommandValidator(nil, []string{"rm -rf /"})
		err := v.ValidateCommand("rm", []string{"-rf", "/"})
		assert.Error(t, err)
	})

	t.Run("deny sudo", func(t *testing.T) {
		v := NewCommandValidator(nil, []string{"sudo"})
		err := v.ValidateCommand("sudo", []string{"apt", "update"})
		assert.Error(t, err)
	})

	t.Run("empty command", func(t *testing.T) {
		v := NewCommandValidator(nil, nil)
		err := v.ValidateCommand("", nil)
		assert.Error(t, err)
	})
}

func TestValidateEnvVarName(t *testing.T) {
	t.Run("valid names", func(t *testing.T) {
		validNames := []string{"PATH", "HOME", "MY_VAR", "_private", "var123"}
		for _, name := range validNames {
			err := ValidateEnvVarName(name)
			assert.NoError(t, err, "Expected %s to be valid", name)
		}
	})

	t.Run("invalid names", func(t *testing.T) {
		invalidNames := []string{"", "123start", "has-dash", "has.dot", "has space"}
		for _, name := range invalidNames {
			err := ValidateEnvVarName(name)
			assert.Error(t, err, "Expected %s to be invalid", name)
		}
	})
}

func TestValidatePort(t *testing.T) {
	t.Run("valid ports", func(t *testing.T) {
		validPorts := []int{1, 80, 443, 8080, 65535}
		for _, port := range validPorts {
			err := ValidatePort(port)
			assert.NoError(t, err, "Expected port %d to be valid", port)
		}
	})

	t.Run("invalid ports", func(t *testing.T) {
		invalidPorts := []int{0, -1, 65536, 100000}
		for _, port := range invalidPorts {
			err := ValidatePort(port)
			assert.Error(t, err, "Expected port %d to be invalid", port)
		}
	})
}

func TestValidatePID(t *testing.T) {
	t.Run("valid PIDs", func(t *testing.T) {
		err := ValidatePID(1)
		assert.NoError(t, err)

		err = ValidatePID(12345)
		assert.NoError(t, err)
	})

	t.Run("invalid PIDs", func(t *testing.T) {
		err := ValidatePID(0)
		assert.Error(t, err)

		err = ValidatePID(-1)
		assert.Error(t, err)
	})
}

func TestResolvePath(t *testing.T) {
	homeDir, _ := os.UserHomeDir()
	v := NewPathValidator([]string{homeDir}, nil, true)

	t.Run("resolve valid path", func(t *testing.T) {
		resolved, err := v.ResolvePath(homeDir)
		require.NoError(t, err)
		assert.Equal(t, homeDir, resolved)
	})

	t.Run("resolve invalid path", func(t *testing.T) {
		_, err := v.ResolvePath("/nonexistent/restricted/path")
		assert.Error(t, err)
	})
}
