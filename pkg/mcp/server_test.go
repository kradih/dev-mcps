package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	server := NewServer("test-server", "1.0.0")
	assert.NotNil(t, server)
	assert.Equal(t, "test-server", server.name)
	assert.Equal(t, "1.0.0", server.version)
}

func TestRegisterTool(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	tool := &Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: BuildInputSchema(
			map[string]interface{}{
				"param1": StringProperty("A string parameter"),
			},
			[]string{"param1"},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
			return TextResult("success"), nil
		},
	}

	server.RegisterTool(tool)

	server.mu.RLock()
	defer server.mu.RUnlock()
	assert.Contains(t, server.tools, "test_tool")
}

func TestHandleInitialize(t *testing.T) {
	var output bytes.Buffer
	server := NewServer("test-server", "1.0.0")
	server.SetIO(strings.NewReader(""), &output)

	req := &Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}

	server.handleRequest(context.Background(), req)

	var resp Response
	err := json.Unmarshal(output.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, float64(1), resp.ID)
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "2024-11-05", result["protocolVersion"])
}

func TestHandleToolsList(t *testing.T) {
	var output bytes.Buffer
	server := NewServer("test-server", "1.0.0")
	server.SetIO(strings.NewReader(""), &output)

	tool := &Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: BuildInputSchema(
			map[string]interface{}{
				"param1": StringProperty("A string parameter"),
			},
			[]string{"param1"},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
			return TextResult("success"), nil
		},
	}
	server.RegisterTool(tool)

	req := &Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}

	server.handleRequest(context.Background(), req)

	var resp Response
	err := json.Unmarshal(output.Bytes(), &resp)
	require.NoError(t, err)

	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(map[string]interface{})
	require.True(t, ok)

	tools, ok := result["tools"].([]interface{})
	require.True(t, ok)
	assert.Len(t, tools, 1)
}

func TestHandleToolsCall(t *testing.T) {
	var output bytes.Buffer
	server := NewServer("test-server", "1.0.0")
	server.SetIO(strings.NewReader(""), &output)

	tool := &Tool{
		Name:        "echo",
		Description: "Echo back the input",
		InputSchema: BuildInputSchema(
			map[string]interface{}{
				"message": StringProperty("Message to echo"),
			},
			[]string{"message"},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
			msg, _ := GetStringParam(params, "message", true)
			return TextResult("Echo: " + msg), nil
		},
	}
	server.RegisterTool(tool)

	params, _ := json.Marshal(map[string]interface{}{
		"name":      "echo",
		"arguments": map[string]interface{}{"message": "hello"},
	})

	req := &Request{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params:  params,
	}

	server.handleRequest(context.Background(), req)

	var resp Response
	err := json.Unmarshal(output.Bytes(), &resp)
	require.NoError(t, err)

	assert.Nil(t, resp.Error)
}

func TestTextResult(t *testing.T) {
	result := TextResult("test message")
	assert.Len(t, result.Content, 1)
	assert.Equal(t, "text", result.Content[0].Type)
	assert.Equal(t, "test message", result.Content[0].Text)
	assert.False(t, result.IsError)
}

func TestJSONResult(t *testing.T) {
	data := map[string]interface{}{
		"key": "value",
		"num": 42,
	}

	result, err := JSONResult(data)
	require.NoError(t, err)
	assert.Len(t, result.Content, 1)
	assert.Contains(t, result.Content[0].Text, "key")
	assert.Contains(t, result.Content[0].Text, "value")
}

func TestErrorResult(t *testing.T) {
	result := ErrorResult(assert.AnError)
	assert.True(t, result.IsError)
	assert.Len(t, result.Content, 1)
}

func TestBuildInputSchema(t *testing.T) {
	schema := BuildInputSchema(
		map[string]interface{}{
			"name": StringProperty("User name"),
			"age":  IntProperty("User age"),
		},
		[]string{"name"},
	)

	assert.Equal(t, "object", schema["type"])
	assert.Contains(t, schema["required"], "name")

	props, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, props, "name")
	assert.Contains(t, props, "age")
}
