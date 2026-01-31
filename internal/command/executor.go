package command

import (
	"bytes"
	"context"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/local-mcps/dev-mcps/config"
)

type AsyncCommand struct {
	ID        string
	Cmd       *exec.Cmd
	Stdout    *bytes.Buffer
	Stderr    *bytes.Buffer
	StartTime time.Time
	EndTime   time.Time
	Status    string
	ExitCode  int
	Cancel    context.CancelFunc
}

type Executor struct {
	config        *config.CommandConfig
	asyncCommands sync.Map
}

func NewExecutor(cfg *config.CommandConfig) *Executor {
	return &Executor{
		config: cfg,
	}
}

func (e *Executor) RunSync(ctx context.Context, command string, args []string, cwd string, env map[string]string, timeoutSeconds int) (*CommandResult, error) {
	if timeoutSeconds <= 0 {
		timeoutSeconds = e.config.DefaultTimeoutSeconds
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)

	if cwd != "" {
		cmd.Dir = cwd
	} else if e.config.WorkingDirectory != "" {
		cmd.Dir = e.config.WorkingDirectory
	}

	if len(env) > 0 {
		cmd.Env = cmd.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	result := &CommandResult{
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		DurationMs: duration.Milliseconds(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
			result.ExitCode = -1
			result.Stderr = "command timed out"
		} else {
			result.ExitCode = -1
			result.Stderr = err.Error()
		}
	}

	if len(result.Stdout) > e.config.MaxOutputSizeBytes {
		result.Stdout = result.Stdout[:e.config.MaxOutputSizeBytes] + "\n... (truncated)"
	}
	if len(result.Stderr) > e.config.MaxOutputSizeBytes {
		result.Stderr = result.Stderr[:e.config.MaxOutputSizeBytes] + "\n... (truncated)"
	}

	return result, nil
}

func (e *Executor) RunAsync(command string, args []string, cwd string, env map[string]string) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, command, args...)

	if cwd != "" {
		cmd.Dir = cwd
	} else if e.config.WorkingDirectory != "" {
		cmd.Dir = e.config.WorkingDirectory
	}

	if len(env) > 0 {
		cmd.Env = cmd.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	asyncCmd := &AsyncCommand{
		ID:        uuid.New().String(),
		Cmd:       cmd,
		Stdout:    &stdout,
		Stderr:    &stderr,
		StartTime: time.Now(),
		Status:    "running",
		Cancel:    cancel,
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return "", err
	}

	e.asyncCommands.Store(asyncCmd.ID, asyncCmd)

	go func() {
		err := cmd.Wait()
		asyncCmd.EndTime = time.Now()

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				asyncCmd.ExitCode = exitErr.ExitCode()
				asyncCmd.Status = "failed"
			} else {
				asyncCmd.ExitCode = -1
				asyncCmd.Status = "cancelled"
			}
		} else {
			asyncCmd.ExitCode = 0
			asyncCmd.Status = "completed"
		}
	}()

	return asyncCmd.ID, nil
}

func (e *Executor) GetStatus(commandID string) (*AsyncCommand, bool) {
	if v, ok := e.asyncCommands.Load(commandID); ok {
		return v.(*AsyncCommand), true
	}
	return nil, false
}

func (e *Executor) CancelCommand(commandID string) bool {
	if v, ok := e.asyncCommands.Load(commandID); ok {
		asyncCmd := v.(*AsyncCommand)
		if asyncCmd.Status == "running" {
			asyncCmd.Cancel()
			asyncCmd.Status = "cancelled"
			return true
		}
	}
	return false
}

type CommandResult struct {
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	DurationMs int64  `json:"duration_ms"`
	CommandID  string `json:"command_id,omitempty"`
}
