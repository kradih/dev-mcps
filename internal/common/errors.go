package common

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound          = errors.New("not found")
	ErrPermissionDenied  = errors.New("permission denied")
	ErrInvalidPath       = errors.New("invalid path")
	ErrPathNotAllowed    = errors.New("path not allowed")
	ErrCommandDenied     = errors.New("command denied")
	ErrTimeout           = errors.New("operation timed out")
	ErrFileTooLarge      = errors.New("file too large")
	ErrInvalidInput      = errors.New("invalid input")
	ErrOperationFailed   = errors.New("operation failed")
	ErrNotImplemented    = errors.New("not implemented")
	ErrProcessNotFound   = errors.New("process not found")
	ErrNotADirectory     = errors.New("not a directory")
	ErrNotAFile          = errors.New("not a file")
	ErrAlreadyExists     = errors.New("already exists")
	ErrDirectoryNotEmpty = errors.New("directory not empty")
)

type MCPError struct {
	Code    string
	Message string
	Cause   error
}

func (e *MCPError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *MCPError) Unwrap() error {
	return e.Cause
}

func NewMCPError(code, message string, cause error) *MCPError {
	return &MCPError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsPermissionDenied(err error) bool {
	return errors.Is(err, ErrPermissionDenied)
}

func IsPathNotAllowed(err error) bool {
	return errors.Is(err, ErrPathNotAllowed)
}
