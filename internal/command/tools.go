package command

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/local-mcps/dev-mcps/pkg/mcp"
)

func (s *Server) runCommandTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "run_command",
		Description: "Execute a shell command synchronously",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"command":         mcp.StringProperty("Command to execute"),
				"args":            mcp.ArrayProperty("string", "Command arguments"),
				"cwd":             mcp.StringProperty("Working directory"),
				"env":             mcp.MapProperty("Environment variables"),
				"timeout_seconds": mcp.IntProperty("Command timeout in seconds"),
			},
			[]string{"command"},
		),
		Handler: s.handleRunCommand,
	}
}

func (s *Server) handleRunCommand(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	command, err := mcp.GetStringParam(params, "command", true)
	if err != nil {
		return nil, err
	}

	args, _ := mcp.GetStringArrayParam(params, "args", false)
	cwd, _ := mcp.GetStringParam(params, "cwd", false)
	env, _ := mcp.GetMapParam(params, "env", false)
	timeout, _ := mcp.GetIntParam(params, "timeout_seconds", false, s.config.DefaultTimeoutSeconds)

	if err := s.validator.ValidateCommand(command, args); err != nil {
		return nil, err
	}

	result, err := s.executor.RunSync(ctx, command, args, cwd, env, timeout)
	if err != nil {
		return nil, err
	}

	return mcp.JSONResult(result)
}

func (s *Server) runCommandAsyncTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "run_command_async",
		Description: "Execute a command asynchronously",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"command": mcp.StringProperty("Command to execute"),
				"args":    mcp.ArrayProperty("string", "Command arguments"),
				"cwd":     mcp.StringProperty("Working directory"),
				"env":     mcp.MapProperty("Environment variables"),
			},
			[]string{"command"},
		),
		Handler: s.handleRunCommandAsync,
	}
}

func (s *Server) handleRunCommandAsync(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	command, err := mcp.GetStringParam(params, "command", true)
	if err != nil {
		return nil, err
	}

	args, _ := mcp.GetStringArrayParam(params, "args", false)
	cwd, _ := mcp.GetStringParam(params, "cwd", false)
	env, _ := mcp.GetMapParam(params, "env", false)

	if err := s.validator.ValidateCommand(command, args); err != nil {
		return nil, err
	}

	commandID, err := s.executor.RunAsync(command, args, cwd, env)
	if err != nil {
		return nil, err
	}

	return mcp.JSONResult(map[string]interface{}{
		"command_id": commandID,
		"status":     "running",
	})
}

func (s *Server) getCommandStatusTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_command_status",
		Description: "Get status of an async command",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"command_id": mcp.StringProperty("ID of the async command"),
			},
			[]string{"command_id"},
		),
		Handler: s.handleGetCommandStatus,
	}
}

func (s *Server) handleGetCommandStatus(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	commandID, err := mcp.GetStringParam(params, "command_id", true)
	if err != nil {
		return nil, err
	}

	asyncCmd, found := s.executor.GetStatus(commandID)
	if !found {
		return nil, fmt.Errorf("command not found: %s", commandID)
	}

	result := map[string]interface{}{
		"command_id": asyncCmd.ID,
		"status":     asyncCmd.Status,
		"exit_code":  asyncCmd.ExitCode,
		"stdout":     asyncCmd.Stdout.String(),
		"stderr":     asyncCmd.Stderr.String(),
	}

	if !asyncCmd.EndTime.IsZero() {
		result["duration_ms"] = asyncCmd.EndTime.Sub(asyncCmd.StartTime).Milliseconds()
	} else {
		result["elapsed_ms"] = asyncCmd.StartTime.Unix()
	}

	return mcp.JSONResult(result)
}

func (s *Server) cancelCommandTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "cancel_command",
		Description: "Cancel a running async command",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"command_id": mcp.StringProperty("ID of the command to cancel"),
			},
			[]string{"command_id"},
		),
		Handler: s.handleCancelCommand,
	}
}

func (s *Server) handleCancelCommand(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	commandID, err := mcp.GetStringParam(params, "command_id", true)
	if err != nil {
		return nil, err
	}

	if s.executor.CancelCommand(commandID) {
		return mcp.TextResult(fmt.Sprintf("Command %s cancelled", commandID)), nil
	}

	return nil, fmt.Errorf("command not found or already completed: %s", commandID)
}

func (s *Server) runScriptTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "run_script",
		Description: "Execute a script file",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"path":        mcp.StringProperty("Path to script file"),
				"interpreter": mcp.StringProperty("Interpreter to use (bash, python, etc.)"),
				"args":        mcp.ArrayProperty("string", "Script arguments"),
				"cwd":         mcp.StringProperty("Working directory"),
			},
			[]string{"path"},
		),
		Handler: s.handleRunScript,
	}
}

func (s *Server) handleRunScript(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	path, err := mcp.GetStringParam(params, "path", true)
	if err != nil {
		return nil, err
	}

	interpreter, _ := mcp.GetStringParam(params, "interpreter", false)
	args, _ := mcp.GetStringArrayParam(params, "args", false)
	cwd, _ := mcp.GetStringParam(params, "cwd", false)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("script not found: %s", path)
	}

	if interpreter == "" {
		interpreter = s.config.DefaultShell
	}

	scriptArgs := append([]string{path}, args...)

	result, err := s.executor.RunSync(ctx, interpreter, scriptArgs, cwd, nil, s.config.DefaultTimeoutSeconds)
	if err != nil {
		return nil, err
	}

	return mcp.JSONResult(result)
}

func (s *Server) getShellInfoTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_shell_info",
		Description: "Get information about available shells",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{},
			[]string{},
		),
		Handler: s.handleGetShellInfo,
	}
}

func (s *Server) handleGetShellInfo(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	result := map[string]interface{}{
		"default_shell": s.config.DefaultShell,
		"os":            runtime.GOOS,
		"arch":          runtime.GOARCH,
	}

	var availableShells []string
	var shellVersions = make(map[string]string)

	shells := []string{"/bin/bash", "/bin/zsh", "/bin/sh"}
	if runtime.GOOS == "windows" {
		shells = []string{"cmd.exe", "powershell.exe"}
	}

	for _, shell := range shells {
		if _, err := exec.LookPath(shell); err == nil {
			availableShells = append(availableShells, shell)

			versionCmd := exec.Command(shell, "--version")
			if output, err := versionCmd.Output(); err == nil {
				lines := strings.Split(string(output), "\n")
				if len(lines) > 0 {
					shellVersions[shell] = strings.TrimSpace(lines[0])
				}
			}
		}
	}

	result["available_shells"] = availableShells
	result["shell_versions"] = shellVersions

	return mcp.JSONResult(result)
}
