package common

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type PathValidator struct {
	AllowedPaths   []string
	DeniedPaths    []string
	FollowSymlinks bool
}

func NewPathValidator(allowed, denied []string, followSymlinks bool) *PathValidator {
	expandedAllowed := make([]string, len(allowed))
	expandedDenied := make([]string, len(denied))

	for i, p := range allowed {
		expandedAllowed[i] = os.ExpandEnv(p)
	}
	for i, p := range denied {
		expandedDenied[i] = os.ExpandEnv(p)
	}

	return &PathValidator{
		AllowedPaths:   expandedAllowed,
		DeniedPaths:    expandedDenied,
		FollowSymlinks: followSymlinks,
	}
}

func (v *PathValidator) ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("%w: empty path", ErrInvalidPath)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("%w: cannot resolve absolute path: %v", ErrInvalidPath, err)
	}

	cleanPath := filepath.Clean(absPath)

	if !v.FollowSymlinks {
		info, err := os.Lstat(cleanPath)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: symlinks not allowed", ErrPathNotAllowed)
		}
	}

	for _, denied := range v.DeniedPaths {
		if strings.HasPrefix(cleanPath, denied) {
			return fmt.Errorf("%w: path is in denied list", ErrPathNotAllowed)
		}
	}

	if len(v.AllowedPaths) == 0 {
		return nil
	}

	for _, allowed := range v.AllowedPaths {
		if strings.HasPrefix(cleanPath, allowed) {
			return nil
		}
	}

	return fmt.Errorf("%w: path not in allowed list", ErrPathNotAllowed)
}

func (v *PathValidator) ResolvePath(path string) (string, error) {
	if err := v.ValidatePath(path); err != nil {
		return "", err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return filepath.Clean(absPath), nil
}

type CommandValidator struct {
	AllowedCommands []string
	DeniedCommands  []string
}

func NewCommandValidator(allowed, denied []string) *CommandValidator {
	return &CommandValidator{
		AllowedCommands: allowed,
		DeniedCommands:  denied,
	}
}

func (v *CommandValidator) ValidateCommand(command string, args []string) error {
	if command == "" {
		return fmt.Errorf("%w: empty command", ErrInvalidInput)
	}

	fullCmd := command
	if len(args) > 0 {
		fullCmd = command + " " + strings.Join(args, " ")
	}

	for _, denied := range v.DeniedCommands {
		if strings.Contains(fullCmd, denied) {
			return fmt.Errorf("%w: command matches denied pattern: %s", ErrCommandDenied, denied)
		}
	}

	if len(v.AllowedCommands) == 0 {
		return nil
	}

	for _, allowed := range v.AllowedCommands {
		if strings.HasPrefix(command, allowed) {
			return nil
		}
	}

	return fmt.Errorf("%w: command not in allowed list", ErrCommandDenied)
}

var envVarNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func ValidateEnvVarName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty environment variable name", ErrInvalidInput)
	}
	if !envVarNameRegex.MatchString(name) {
		return fmt.Errorf("%w: invalid environment variable name: %s", ErrInvalidInput, name)
	}
	return nil
}

func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%w: port must be between 1 and 65535", ErrInvalidInput)
	}
	return nil
}

func ValidatePID(pid int) error {
	if pid < 1 {
		return fmt.Errorf("%w: PID must be positive", ErrInvalidInput)
	}
	return nil
}
