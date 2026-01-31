package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Global      GlobalConfig      `yaml:"global"`
	Filesystem  FilesystemConfig  `yaml:"filesystem"`
	Command     CommandConfig     `yaml:"command"`
	Web         WebConfig         `yaml:"web"`
	Environment EnvironmentConfig `yaml:"environment"`
	Git         GitConfig         `yaml:"git"`
	Process     ProcessConfig     `yaml:"process"`
}

type GlobalConfig struct {
	LogLevel  string `yaml:"log_level"`
	LogFormat string `yaml:"log_format"`
	Transport string `yaml:"transport"`
	HTTPPort  int    `yaml:"http_port"`
}

type FilesystemConfig struct {
	Enabled        bool     `yaml:"enabled"`
	AllowedPaths   []string `yaml:"allowed_paths"`
	DeniedPaths    []string `yaml:"denied_paths"`
	MaxFileSizeMB  int      `yaml:"max_file_size_mb"`
	FollowSymlinks bool     `yaml:"follow_symlinks"`
}

type CommandConfig struct {
	Enabled               bool     `yaml:"enabled"`
	DefaultShell          string   `yaml:"default_shell"`
	DefaultTimeoutSeconds int      `yaml:"default_timeout_seconds"`
	MaxOutputSizeBytes    int      `yaml:"max_output_size_bytes"`
	AllowedCommands       []string `yaml:"allowed_commands"`
	DeniedCommands        []string `yaml:"denied_commands"`
	WorkingDirectory      string   `yaml:"working_directory"`
}

type WebConfig struct {
	Enabled              bool     `yaml:"enabled"`
	UserAgent            string   `yaml:"user_agent"`
	DefaultTimeoutSeconds int      `yaml:"default_timeout_seconds"`
	MaxResponseSizeBytes int      `yaml:"max_response_size_bytes"`
	FollowRedirects      bool     `yaml:"follow_redirects"`
	MaxRedirects         int      `yaml:"max_redirects"`
	ProxyURL             string   `yaml:"proxy_url"`
	AllowedDomains       []string `yaml:"allowed_domains"`
	DeniedDomains        []string `yaml:"denied_domains"`
	EnableJavascript     bool     `yaml:"enable_javascript"`
}

type EnvironmentConfig struct {
	Enabled            bool     `yaml:"enabled"`
	ExposeAllEnv       bool     `yaml:"expose_all_env"`
	AllowedEnvPrefixes []string `yaml:"allowed_env_prefixes"`
	DeniedEnvPatterns  []string `yaml:"denied_env_patterns"`
}

type GitConfig struct {
	Enabled             bool     `yaml:"enabled"`
	AllowedRepositories []string `yaml:"allowed_repositories"`
	AllowPush           bool     `yaml:"allow_push"`
	AllowForcePush      bool     `yaml:"allow_force_push"`
	DefaultAuthorName   string   `yaml:"default_author_name"`
	DefaultAuthorEmail  string   `yaml:"default_author_email"`
	SignCommits         bool     `yaml:"sign_commits"`
}

type ProcessConfig struct {
	Enabled            bool     `yaml:"enabled"`
	AllowKill          bool     `yaml:"allow_kill"`
	AllowedKillUsers   []string `yaml:"allowed_kill_users"`
	DeniedProcessNames []string `yaml:"denied_process_names"`
	MaxListResults     int      `yaml:"max_list_results"`
}

func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()

	return &Config{
		Global: GlobalConfig{
			LogLevel:  "info",
			LogFormat: "json",
			Transport: "stdio",
			HTTPPort:  8080,
		},
		Filesystem: FilesystemConfig{
			Enabled:        true,
			AllowedPaths:   []string{homeDir},
			DeniedPaths:    []string{filepath.Join(homeDir, ".ssh"), filepath.Join(homeDir, ".gnupg")},
			MaxFileSizeMB:  50,
			FollowSymlinks: false,
		},
		Command: CommandConfig{
			Enabled:               true,
			DefaultShell:          "/bin/bash",
			DefaultTimeoutSeconds: 300,
			MaxOutputSizeBytes:    10485760,
			AllowedCommands:       []string{},
			DeniedCommands:        []string{"rm -rf /", "sudo"},
			WorkingDirectory:      homeDir,
		},
		Web: WebConfig{
			Enabled:              true,
			UserAgent:            "LocalMCP-WebBrowser/1.0",
			DefaultTimeoutSeconds: 30,
			MaxResponseSizeBytes: 52428800,
			FollowRedirects:      true,
			MaxRedirects:         10,
			AllowedDomains:       []string{},
			DeniedDomains:        []string{},
			EnableJavascript:     false,
		},
		Environment: EnvironmentConfig{
			Enabled:            true,
			ExposeAllEnv:       false,
			AllowedEnvPrefixes: []string{"PATH", "HOME", "USER", "GOPATH", "NODE_", "NPM_"},
			DeniedEnvPatterns:  []string{".*_KEY$", ".*_SECRET$", ".*_TOKEN$", ".*_PASSWORD$"},
		},
		Git: GitConfig{
			Enabled:             true,
			AllowedRepositories: []string{homeDir},
			AllowPush:           true,
			AllowForcePush:      false,
			DefaultAuthorName:   "MCP Agent",
			DefaultAuthorEmail:  "mcp@localhost",
			SignCommits:         false,
		},
		Process: ProcessConfig{
			Enabled:            true,
			AllowKill:          true,
			AllowedKillUsers:   []string{os.Getenv("USER")},
			DeniedProcessNames: []string{"init", "systemd", "launchd"},
			MaxListResults:     1000,
		},
	}
}

func LoadConfig(path string) (*Config, error) {
	config := DefaultConfig()

	if path == "" {
		configDir, err := os.UserConfigDir()
		if err == nil {
			path = filepath.Join(configDir, "local-mcps", "config.yaml")
		}
	}

	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			if err := yaml.Unmarshal(data, config); err != nil {
				return nil, err
			}
		}
	}

	applyEnvOverrides(config)

	return config, nil
}

func applyEnvOverrides(config *Config) {
	if v := os.Getenv("LOCAL_MCP_LOG_LEVEL"); v != "" {
		config.Global.LogLevel = v
	}
	if v := os.Getenv("LOCAL_MCP_LOG_FORMAT"); v != "" {
		config.Global.LogFormat = v
	}
	if v := os.Getenv("LOCAL_MCP_TRANSPORT"); v != "" {
		config.Global.Transport = v
	}
	if v := os.Getenv("LOCAL_MCP_HTTP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			config.Global.HTTPPort = port
		}
	}
	if v := os.Getenv("LOCAL_MCP_FILESYSTEM_ALLOWED_PATHS"); v != "" {
		config.Filesystem.AllowedPaths = strings.Split(v, ":")
	}
	if v := os.Getenv("LOCAL_MCP_COMMAND_TIMEOUT"); v != "" {
		if timeout, err := strconv.Atoi(v); err == nil {
			config.Command.DefaultTimeoutSeconds = timeout
		}
	}
}

func (c *Config) ExpandPaths() {
	for i, p := range c.Filesystem.AllowedPaths {
		c.Filesystem.AllowedPaths[i] = os.ExpandEnv(p)
	}
	for i, p := range c.Filesystem.DeniedPaths {
		c.Filesystem.DeniedPaths[i] = os.ExpandEnv(p)
	}
	c.Command.WorkingDirectory = os.ExpandEnv(c.Command.WorkingDirectory)
	for i, p := range c.Git.AllowedRepositories {
		c.Git.AllowedRepositories[i] = os.ExpandEnv(p)
	}
}
