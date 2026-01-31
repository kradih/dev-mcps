package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStringParam(t *testing.T) {
	params := map[string]interface{}{
		"name":  "test",
		"empty": "",
	}

	t.Run("existing required param", func(t *testing.T) {
		val, err := GetStringParam(params, "name", true)
		require.NoError(t, err)
		assert.Equal(t, "test", val)
	})

	t.Run("missing required param", func(t *testing.T) {
		_, err := GetStringParam(params, "missing", true)
		assert.Error(t, err)
	})

	t.Run("missing optional param", func(t *testing.T) {
		val, err := GetStringParam(params, "missing", false)
		require.NoError(t, err)
		assert.Equal(t, "", val)
	})

	t.Run("wrong type", func(t *testing.T) {
		params["num"] = 123
		_, err := GetStringParam(params, "num", true)
		assert.Error(t, err)
	})
}

func TestGetIntParam(t *testing.T) {
	params := map[string]interface{}{
		"count":    float64(42),
		"zero":     float64(0),
		"negative": float64(-10),
	}

	t.Run("existing required param", func(t *testing.T) {
		val, err := GetIntParam(params, "count", true, 0)
		require.NoError(t, err)
		assert.Equal(t, 42, val)
	})

	t.Run("missing required param", func(t *testing.T) {
		_, err := GetIntParam(params, "missing", true, 0)
		assert.Error(t, err)
	})

	t.Run("missing optional param with default", func(t *testing.T) {
		val, err := GetIntParam(params, "missing", false, 100)
		require.NoError(t, err)
		assert.Equal(t, 100, val)
	})

	t.Run("zero value", func(t *testing.T) {
		val, err := GetIntParam(params, "zero", true, 0)
		require.NoError(t, err)
		assert.Equal(t, 0, val)
	})

	t.Run("negative value", func(t *testing.T) {
		val, err := GetIntParam(params, "negative", true, 0)
		require.NoError(t, err)
		assert.Equal(t, -10, val)
	})
}

func TestGetBoolParam(t *testing.T) {
	params := map[string]interface{}{
		"enabled":  true,
		"disabled": false,
	}

	t.Run("true value", func(t *testing.T) {
		val, err := GetBoolParam(params, "enabled", false)
		require.NoError(t, err)
		assert.True(t, val)
	})

	t.Run("false value", func(t *testing.T) {
		val, err := GetBoolParam(params, "disabled", true)
		require.NoError(t, err)
		assert.False(t, val)
	})

	t.Run("missing with default true", func(t *testing.T) {
		val, err := GetBoolParam(params, "missing", true)
		require.NoError(t, err)
		assert.True(t, val)
	})

	t.Run("missing with default false", func(t *testing.T) {
		val, err := GetBoolParam(params, "missing", false)
		require.NoError(t, err)
		assert.False(t, val)
	})
}

func TestGetStringArrayParam(t *testing.T) {
	params := map[string]interface{}{
		"items": []interface{}{"a", "b", "c"},
		"empty": []interface{}{},
	}

	t.Run("existing array", func(t *testing.T) {
		val, err := GetStringArrayParam(params, "items", true)
		require.NoError(t, err)
		assert.Equal(t, []string{"a", "b", "c"}, val)
	})

	t.Run("empty array", func(t *testing.T) {
		val, err := GetStringArrayParam(params, "empty", true)
		require.NoError(t, err)
		assert.Empty(t, val)
	})

	t.Run("missing required", func(t *testing.T) {
		_, err := GetStringArrayParam(params, "missing", true)
		assert.Error(t, err)
	})

	t.Run("missing optional", func(t *testing.T) {
		val, err := GetStringArrayParam(params, "missing", false)
		require.NoError(t, err)
		assert.Nil(t, val)
	})
}

func TestGetMapParam(t *testing.T) {
	params := map[string]interface{}{
		"headers": map[string]interface{}{
			"Content-Type": "application/json",
			"Accept":       "text/html",
		},
		"empty": map[string]interface{}{},
	}

	t.Run("existing map", func(t *testing.T) {
		val, err := GetMapParam(params, "headers", true)
		require.NoError(t, err)
		assert.Equal(t, "application/json", val["Content-Type"])
		assert.Equal(t, "text/html", val["Accept"])
	})

	t.Run("empty map", func(t *testing.T) {
		val, err := GetMapParam(params, "empty", true)
		require.NoError(t, err)
		assert.Empty(t, val)
	})

	t.Run("missing required", func(t *testing.T) {
		_, err := GetMapParam(params, "missing", true)
		assert.Error(t, err)
	})

	t.Run("missing optional", func(t *testing.T) {
		val, err := GetMapParam(params, "missing", false)
		require.NoError(t, err)
		assert.Nil(t, val)
	})
}
