package environment

import (
	"context"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/v3/mem"
	"github.com/local-mcps/dev-mcps/pkg/mcp"
)

func (s *Server) getEnvTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_env",
		Description: "Get an environment variable value",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"name": mcp.StringProperty("Environment variable name"),
			},
			[]string{"name"},
		),
		Handler: s.handleGetEnv,
	}
}

func (s *Server) handleGetEnv(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	name, err := mcp.GetStringParam(params, "name", true)
	if err != nil {
		return nil, err
	}

	if s.isSensitive(name) {
		return mcp.JSONResult(map[string]interface{}{
			"name":     name,
			"value":    "",
			"exists":   false,
			"filtered": true,
		})
	}

	if !s.isAllowed(name) {
		return mcp.JSONResult(map[string]interface{}{
			"name":     name,
			"value":    "",
			"exists":   false,
			"filtered": true,
		})
	}

	if value, ok := s.sessionEnv[name]; ok {
		return mcp.JSONResult(map[string]interface{}{
			"name":   name,
			"value":  value,
			"exists": true,
		})
	}

	value, exists := os.LookupEnv(name)
	return mcp.JSONResult(map[string]interface{}{
		"name":   name,
		"value":  value,
		"exists": exists,
	})
}

func (s *Server) setEnvTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "set_env",
		Description: "Set an environment variable (session-scoped)",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"name":  mcp.StringProperty("Variable name"),
				"value": mcp.StringProperty("Variable value"),
			},
			[]string{"name", "value"},
		),
		Handler: s.handleSetEnv,
	}
}

func (s *Server) handleSetEnv(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	name, err := mcp.GetStringParam(params, "name", true)
	if err != nil {
		return nil, err
	}

	value, err := mcp.GetStringParam(params, "value", true)
	if err != nil {
		return nil, err
	}

	s.sessionEnv[name] = value

	return mcp.JSONResult(map[string]interface{}{
		"name":  name,
		"value": value,
		"set":   true,
	})
}

func (s *Server) listEnvTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "list_env",
		Description: "List all environment variables",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"filter_prefix": mcp.StringProperty("Filter by prefix"),
			},
			[]string{},
		),
		Handler: s.handleListEnv,
	}
}

func (s *Server) handleListEnv(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	filterPrefix, _ := mcp.GetStringParam(params, "filter_prefix", false)

	var variables []map[string]string
	filteredCount := 0

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		name, value := parts[0], parts[1]

		if filterPrefix != "" && !strings.HasPrefix(name, filterPrefix) {
			continue
		}

		if s.isSensitive(name) {
			filteredCount++
			continue
		}

		if !s.config.ExposeAllEnv && !s.isAllowed(name) {
			filteredCount++
			continue
		}

		variables = append(variables, map[string]string{
			"name":  name,
			"value": value,
		})
	}

	for name, value := range s.sessionEnv {
		if filterPrefix != "" && !strings.HasPrefix(name, filterPrefix) {
			continue
		}
		variables = append(variables, map[string]string{
			"name":  name,
			"value": value,
		})
	}

	return mcp.JSONResult(map[string]interface{}{
		"variables":      variables,
		"total_count":    len(variables),
		"filtered_count": filteredCount,
	})
}

func (s *Server) unsetEnvTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "unset_env",
		Description: "Unset a session environment variable",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"name": mcp.StringProperty("Variable name"),
			},
			[]string{"name"},
		),
		Handler: s.handleUnsetEnv,
	}
}

func (s *Server) handleUnsetEnv(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	name, err := mcp.GetStringParam(params, "name", true)
	if err != nil {
		return nil, err
	}

	if _, ok := s.sessionEnv[name]; ok {
		delete(s.sessionEnv, name)
		return mcp.JSONResult(map[string]interface{}{
			"name":  name,
			"unset": true,
		})
	}

	return mcp.JSONResult(map[string]interface{}{
		"name":  name,
		"unset": false,
		"error": "variable not set in session",
	})
}

func (s *Server) getSystemInfoTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_system_info",
		Description: "Get system information",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{},
			[]string{},
		),
		Handler: s.handleGetSystemInfo,
	}
}

func (s *Server) handleGetSystemInfo(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	hostname, _ := os.Hostname()
	homeDir, _ := os.UserHomeDir()
	tempDir := os.TempDir()

	result := map[string]interface{}{
		"os":             runtime.GOOS,
		"arch":           runtime.GOARCH,
		"hostname":       hostname,
		"num_cpu":        runtime.NumCPU(),
		"go_version":     runtime.Version(),
		"home_directory": homeDir,
		"temp_directory": tempDir,
	}

	if memInfo, err := mem.VirtualMemory(); err == nil {
		result["total_memory_gb"] = float64(memInfo.Total) / (1024 * 1024 * 1024)
		result["available_memory_gb"] = float64(memInfo.Available) / (1024 * 1024 * 1024)
	}

	return mcp.JSONResult(result)
}

func (s *Server) getUserInfoTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_user_info",
		Description: "Get current user information",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{},
			[]string{},
		),
		Handler: s.handleGetUserInfo,
	}
}

func (s *Server) handleGetUserInfo(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	currentUser, err := user.Current()
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"username":       currentUser.Username,
		"uid":            currentUser.Uid,
		"gid":            currentUser.Gid,
		"home_directory": currentUser.HomeDir,
	}

	if shell := os.Getenv("SHELL"); shell != "" {
		result["shell"] = shell
	}

	if groupIds, err := currentUser.GroupIds(); err == nil {
		result["groups"] = groupIds
	}

	return mcp.JSONResult(result)
}

func (s *Server) getPathInfoTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_path_info",
		Description: "Get PATH and related paths",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{},
			[]string{},
		),
		Handler: s.handleGetPathInfo,
	}
}

func (s *Server) handleGetPathInfo(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	pathEnv := os.Getenv("PATH")
	pathEntries := strings.Split(pathEnv, string(os.PathListSeparator))

	pwd, _ := os.Getwd()
	homeDir, _ := os.UserHomeDir()

	result := map[string]interface{}{
		"path_entries": pathEntries,
		"home":         homeDir,
		"pwd":          pwd,
		"tmpdir":       os.TempDir(),
	}

	if gopath := os.Getenv("GOPATH"); gopath != "" {
		result["gopath"] = gopath
	}
	if goroot := os.Getenv("GOROOT"); goroot != "" {
		result["goroot"] = goroot
	}

	return mcp.JSONResult(result)
}

func (s *Server) expandPathTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "expand_path",
		Description: "Expand path with variables (e.g., ~, $HOME)",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"path": mcp.StringProperty("Path to expand"),
			},
			[]string{"path"},
		),
		Handler: s.handleExpandPath,
	}
}

func (s *Server) handleExpandPath(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	path, err := mcp.GetStringParam(params, "path", true)
	if err != nil {
		return nil, err
	}

	expanded := path

	if strings.HasPrefix(expanded, "~") {
		homeDir, _ := os.UserHomeDir()
		expanded = homeDir + expanded[1:]
	}

	expanded = os.ExpandEnv(expanded)

	absPath, _ := filepath.Abs(expanded)
	expanded = absPath

	info, err := os.Stat(expanded)
	exists := err == nil
	isDir := exists && info.IsDir()

	return mcp.JSONResult(map[string]interface{}{
		"original":     path,
		"expanded":     expanded,
		"exists":       exists,
		"is_directory": isDir,
	})
}

func (s *Server) isSensitive(name string) bool {
	patterns := s.config.DeniedEnvPatterns
	if len(patterns) == 0 {
		patterns = []string{
			`(?i).*_KEY$`,
			`(?i).*_SECRET$`,
			`(?i).*_TOKEN$`,
			`(?i).*_PASSWORD$`,
			`(?i)^AWS_`,
			`(?i)^GITHUB_TOKEN`,
		}
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, name); matched {
			return true
		}
	}
	return false
}

func (s *Server) isAllowed(name string) bool {
	if s.config.ExposeAllEnv {
		return true
	}

	for _, prefix := range s.config.AllowedEnvPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
