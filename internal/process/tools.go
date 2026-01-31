package process

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	gopsProcess "github.com/shirou/gopsutil/v3/process"

	"github.com/local-mcps/dev-mcps/internal/common"
	"github.com/local-mcps/dev-mcps/pkg/mcp"
)

type ProcessInfo struct {
	PID        int32   `json:"pid"`
	Name       string  `json:"name"`
	Command    string  `json:"command"`
	User       string  `json:"user"`
	CPUPercent float64 `json:"cpu_percent"`
	MemoryMB   float64 `json:"memory_mb"`
	Status     string  `json:"status"`
	StartTime  string  `json:"start_time"`
}

func (s *Server) listProcessesTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "list_processes",
		Description: "List running processes",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"filter_name": mcp.StringProperty("Filter by process name"),
				"filter_user": mcp.StringProperty("Filter by user"),
			},
			[]string{},
		),
		Handler: s.handleListProcesses,
	}
}

func (s *Server) handleListProcesses(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	filterName, _ := mcp.GetStringParam(params, "filter_name", false)
	filterUser, _ := mcp.GetStringParam(params, "filter_user", false)

	processes, err := gopsProcess.Processes()
	if err != nil {
		return nil, err
	}

	var result []ProcessInfo
	maxResults := s.config.MaxListResults
	if maxResults <= 0 {
		maxResults = 1000
	}

	for _, p := range processes {
		if len(result) >= maxResults {
			break
		}

		name, _ := p.Name()
		if filterName != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(filterName)) {
			continue
		}

		username, _ := p.Username()
		if filterUser != "" && username != filterUser {
			continue
		}

		cmdline, _ := p.Cmdline()
		cpuPercent, _ := p.CPUPercent()
		memInfo, _ := p.MemoryInfo()
		status, _ := p.Status()
		createTime, _ := p.CreateTime()

		memMB := float64(0)
		if memInfo != nil {
			memMB = float64(memInfo.RSS) / (1024 * 1024)
		}

		startTimeStr := ""
		if createTime > 0 {
			startTimeStr = time.UnixMilli(createTime).Format(time.RFC3339)
		}

		result = append(result, ProcessInfo{
			PID:        p.Pid,
			Name:       name,
			Command:    cmdline,
			User:       username,
			CPUPercent: cpuPercent,
			MemoryMB:   memMB,
			Status:     strings.Join(status, ","),
			StartTime:  startTimeStr,
		})
	}

	return mcp.JSONResult(map[string]interface{}{
		"processes":   result,
		"total_count": len(result),
	})
}

func (s *Server) getProcessInfoTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_process_info",
		Description: "Get detailed process information",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"pid": mcp.IntProperty("Process ID"),
			},
			[]string{"pid"},
		),
		Handler: s.handleGetProcessInfo,
	}
}

func (s *Server) handleGetProcessInfo(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	pid, err := mcp.GetIntParam(params, "pid", true, 0)
	if err != nil {
		return nil, err
	}

	if err := common.ValidatePID(pid); err != nil {
		return nil, err
	}

	p, err := gopsProcess.NewProcess(int32(pid))
	if err != nil {
		return nil, fmt.Errorf("%w: %d", common.ErrProcessNotFound, pid)
	}

	name, _ := p.Name()
	cmdline, _ := p.Cmdline()
	cmdlineSlice, _ := p.CmdlineSlice()
	username, _ := p.Username()
	status, _ := p.Status()
	cpuPercent, _ := p.CPUPercent()
	memInfo, _ := p.MemoryInfo()
	memPercent, _ := p.MemoryPercent()
	numThreads, _ := p.NumThreads()
	cwd, _ := p.Cwd()
	createTime, _ := p.CreateTime()
	ppid, _ := p.Ppid()

	memMB := float64(0)
	if memInfo != nil {
		memMB = float64(memInfo.RSS) / (1024 * 1024)
	}

	startTimeStr := ""
	if createTime > 0 {
		startTimeStr = time.UnixMilli(createTime).Format(time.RFC3339)
	}

	result := map[string]interface{}{
		"pid":            pid,
		"ppid":           ppid,
		"name":           name,
		"command":        cmdline,
		"command_line":   cmdlineSlice,
		"user":           username,
		"status":         strings.Join(status, ","),
		"cpu_percent":    cpuPercent,
		"memory_mb":      memMB,
		"memory_percent": memPercent,
		"threads":        numThreads,
		"cwd":            cwd,
		"start_time":     startTimeStr,
	}

	return mcp.JSONResult(result)
}

func (s *Server) killProcessTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "kill_process",
		Description: "Terminate a process",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"pid":    mcp.IntProperty("Process ID"),
				"signal": mcp.StringProperty("Signal to send (default: SIGTERM)"),
			},
			[]string{"pid"},
		),
		Handler: s.handleKillProcess,
	}
}

func (s *Server) handleKillProcess(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	pid, err := mcp.GetIntParam(params, "pid", true, 0)
	if err != nil {
		return nil, err
	}

	signalName, _ := mcp.GetStringParam(params, "signal", false)
	if signalName == "" {
		signalName = "SIGTERM"
	}

	if !s.config.AllowKill {
		return nil, fmt.Errorf("kill is disabled in configuration")
	}

	if err := common.ValidatePID(pid); err != nil {
		return nil, err
	}

	if pid == 1 {
		return nil, fmt.Errorf("cannot kill PID 1")
	}

	p, err := gopsProcess.NewProcess(int32(pid))
	if err != nil {
		return nil, fmt.Errorf("%w: %d", common.ErrProcessNotFound, pid)
	}

	name, _ := p.Name()
	for _, denied := range s.config.DeniedProcessNames {
		if strings.EqualFold(name, denied) {
			return nil, fmt.Errorf("cannot kill protected process: %s", name)
		}
	}

	username, _ := p.Username()
	if len(s.config.AllowedKillUsers) > 0 {
		allowed := false
		for _, u := range s.config.AllowedKillUsers {
			if u == username || u == "$USER" && username == os.Getenv("USER") {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("not allowed to kill processes owned by %s", username)
		}
	}

	var sig syscall.Signal
	switch strings.ToUpper(signalName) {
	case "SIGTERM", "TERM":
		sig = syscall.SIGTERM
	case "SIGKILL", "KILL":
		sig = syscall.SIGKILL
	case "SIGINT", "INT":
		sig = syscall.SIGINT
	case "SIGHUP", "HUP":
		sig = syscall.SIGHUP
	default:
		sig = syscall.SIGTERM
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return nil, err
	}

	if err := process.Signal(sig); err != nil {
		return nil, err
	}

	return mcp.TextResult(fmt.Sprintf("Sent %s to process %d (%s)", signalName, pid, name)), nil
}

func (s *Server) findProcessByPortTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "find_process_by_port",
		Description: "Find process using a specific port",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"port": mcp.IntProperty("Port number"),
			},
			[]string{"port"},
		),
		Handler: s.handleFindProcessByPort,
	}
}

func (s *Server) handleFindProcessByPort(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	port, err := mcp.GetIntParam(params, "port", true, 0)
	if err != nil {
		return nil, err
	}

	if err := common.ValidatePort(port); err != nil {
		return nil, err
	}

	connections, err := net.Connections("all")
	if err != nil {
		return nil, err
	}

	var processes []map[string]interface{}
	seenPIDs := make(map[int32]bool)

	for _, conn := range connections {
		if conn.Laddr.Port == uint32(port) && !seenPIDs[conn.Pid] {
			seenPIDs[conn.Pid] = true

			p, err := gopsProcess.NewProcess(conn.Pid)
			if err != nil {
				continue
			}

			name, _ := p.Name()
			user, _ := p.Username()

			processes = append(processes, map[string]interface{}{
				"pid":      conn.Pid,
				"name":     name,
				"user":     user,
				"protocol": conn.Type,
			})
		}
	}

	return mcp.JSONResult(map[string]interface{}{
		"port":      port,
		"processes": processes,
	})
}

func (s *Server) getResourceUsageTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_resource_usage",
		Description: "Get system resource usage (CPU, memory)",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{},
			[]string{},
		),
		Handler: s.handleGetResourceUsage,
	}
}

func (s *Server) handleGetResourceUsage(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	result := make(map[string]interface{})

	if cpuPercent, err := cpu.Percent(100*time.Millisecond, false); err == nil && len(cpuPercent) > 0 {
		result["cpu"] = map[string]interface{}{
			"usage_percent": cpuPercent[0],
			"cores":         len(cpuPercent),
		}
	}

	if memInfo, err := mem.VirtualMemory(); err == nil {
		result["memory"] = map[string]interface{}{
			"total_gb":      float64(memInfo.Total) / (1024 * 1024 * 1024),
			"used_gb":       float64(memInfo.Used) / (1024 * 1024 * 1024),
			"available_gb":  float64(memInfo.Available) / (1024 * 1024 * 1024),
			"usage_percent": memInfo.UsedPercent,
		}
	}

	if diskInfo, err := disk.Usage("/"); err == nil {
		result["disk"] = map[string]interface{}{
			"total_gb":      float64(diskInfo.Total) / (1024 * 1024 * 1024),
			"used_gb":       float64(diskInfo.Used) / (1024 * 1024 * 1024),
			"available_gb":  float64(diskInfo.Free) / (1024 * 1024 * 1024),
			"usage_percent": diskInfo.UsedPercent,
		}
	}

	if netIO, err := net.IOCounters(false); err == nil && len(netIO) > 0 {
		result["network"] = map[string]interface{}{
			"bytes_sent": netIO[0].BytesSent,
			"bytes_recv": netIO[0].BytesRecv,
		}
	}

	return mcp.JSONResult(result)
}

func (s *Server) waitForProcessTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "wait_for_process",
		Description: "Wait for a process to complete",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"pid":             mcp.IntProperty("Process ID"),
				"timeout_seconds": mcp.IntProperty("Wait timeout"),
			},
			[]string{"pid"},
		),
		Handler: s.handleWaitForProcess,
	}
}

func (s *Server) handleWaitForProcess(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	pid, err := mcp.GetIntParam(params, "pid", true, 0)
	if err != nil {
		return nil, err
	}

	timeout, _ := mcp.GetIntParam(params, "timeout_seconds", false, 60)

	if err := common.ValidatePID(pid); err != nil {
		return nil, err
	}

	startTime := time.Now()
	deadline := startTime.Add(time.Duration(timeout) * time.Second)

	for time.Now().Before(deadline) {
		exists, err := gopsProcess.PidExists(int32(pid))
		if err != nil {
			return nil, err
		}

		if !exists {
			return mcp.JSONResult(map[string]interface{}{
				"pid":         pid,
				"completed":   true,
				"duration_ms": time.Since(startTime).Milliseconds(),
			})
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}

	return mcp.JSONResult(map[string]interface{}{
		"pid":         pid,
		"completed":   false,
		"duration_ms": time.Since(startTime).Milliseconds(),
		"timeout":     true,
	})
}

func (s *Server) startProcessTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "start_process",
		Description: "Start a new background process",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"command": mcp.StringProperty("Command to run"),
				"args":    mcp.ArrayProperty("string", "Command arguments"),
				"cwd":     mcp.StringProperty("Working directory"),
			},
			[]string{"command"},
		),
		Handler: s.handleStartProcess,
	}
}

func (s *Server) handleStartProcess(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	command, err := mcp.GetStringParam(params, "command", true)
	if err != nil {
		return nil, err
	}

	args, _ := mcp.GetStringArrayParam(params, "args", false)
	cwd, _ := mcp.GetStringParam(params, "cwd", false)

	cmd := exec.Command(command, args...)
	if cwd != "" {
		cmd.Dir = cwd
	}

	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go func() {
		cmd.Wait()
	}()

	return mcp.JSONResult(map[string]interface{}{
		"pid":        cmd.Process.Pid,
		"command":    command,
		"started":    true,
		"start_time": time.Now().Format(time.RFC3339),
	})
}
