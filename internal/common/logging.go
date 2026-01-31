package common

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "debug"
	case LogLevelInfo:
		return "info"
	case LogLevelWarn:
		return "warn"
	case LogLevelError:
		return "error"
	default:
		return "unknown"
	}
}

func ParseLogLevel(s string) LogLevel {
	switch s {
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "warn":
		return LogLevelWarn
	case "error":
		return LogLevelError
	default:
		return LogLevelInfo
	}
}

type LogFormat int

const (
	LogFormatJSON LogFormat = iota
	LogFormatText
)

type Logger struct {
	mu       sync.Mutex
	level    LogLevel
	format   LogFormat
	output   io.Writer
	fields   map[string]interface{}
	serverID string
}

func NewLogger(level LogLevel, format LogFormat, output io.Writer, serverID string) *Logger {
	if output == nil {
		output = os.Stderr
	}
	return &Logger{
		level:    level,
		format:   format,
		output:   output,
		fields:   make(map[string]interface{}),
		serverID: serverID,
	}
}

func (l *Logger) WithField(key string, value interface{}) *Logger {
	newLogger := &Logger{
		level:    l.level,
		format:   l.format,
		output:   l.output,
		fields:   make(map[string]interface{}),
		serverID: l.serverID,
	}
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	newLogger.fields[key] = value
	return newLogger
}

func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newLogger := &Logger{
		level:    l.level,
		format:   l.format,
		output:   l.output,
		fields:   make(map[string]interface{}),
		serverID: l.serverID,
	}
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

func (l *Logger) log(level LogLevel, msg string) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     level.String(),
		"message":   msg,
		"server":    l.serverID,
	}

	for k, v := range l.fields {
		entry[k] = v
	}

	if l.format == LogFormatJSON {
		data, _ := json.Marshal(entry)
		fmt.Fprintln(l.output, string(data))
	} else {
		fmt.Fprintf(l.output, "[%s] %s: %s", entry["timestamp"], level.String(), msg)
		for k, v := range l.fields {
			fmt.Fprintf(l.output, " %s=%v", k, v)
		}
		fmt.Fprintln(l.output)
	}
}

func (l *Logger) Debug(msg string) {
	l.log(LogLevelDebug, msg)
}

func (l *Logger) Info(msg string) {
	l.log(LogLevelInfo, msg)
}

func (l *Logger) Warn(msg string) {
	l.log(LogLevelWarn, msg)
}

func (l *Logger) Error(msg string) {
	l.log(LogLevelError, msg)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(LogLevelDebug, fmt.Sprintf(format, args...))
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(LogLevelInfo, fmt.Sprintf(format, args...))
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(LogLevelWarn, fmt.Sprintf(format, args...))
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(LogLevelError, fmt.Sprintf(format, args...))
}

var defaultLogger = NewLogger(LogLevelInfo, LogFormatText, os.Stderr, "default")

func SetDefaultLogger(logger *Logger) {
	defaultLogger = logger
}

func Debug(msg string)                            { defaultLogger.Debug(msg) }
func Info(msg string)                             { defaultLogger.Info(msg) }
func Warn(msg string)                             { defaultLogger.Warn(msg) }
func Error(msg string)                            { defaultLogger.Error(msg) }
func Debugf(format string, args ...interface{})   { defaultLogger.Debugf(format, args...) }
func Infof(format string, args ...interface{})    { defaultLogger.Infof(format, args...) }
func Warnf(format string, args ...interface{})    { defaultLogger.Warnf(format, args...) }
func Errorf(format string, args ...interface{})   { defaultLogger.Errorf(format, args...) }
