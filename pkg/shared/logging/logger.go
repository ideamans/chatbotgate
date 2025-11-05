package logging

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// Level represents log level
type Level int

const (
	// LevelDebug is for debug messages
	LevelDebug Level = iota
	// LevelInfo is for informational messages
	LevelInfo
	// LevelWarn is for warning messages
	LevelWarn
	// LevelError is for error messages
	LevelError
	// LevelFatal is for fatal error messages
	LevelFatal
)

// String returns the string representation of the log level
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel parses a string into a Level
func ParseLevel(s string) Level {
	switch strings.ToLower(s) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	case "fatal":
		return LevelFatal
	default:
		return LevelInfo
	}
}

// Logger is the interface for logging
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Fatal(msg string, args ...interface{})
	WithModule(module string) Logger
}

// SimpleLogger is a basic logger implementation
type SimpleLogger struct {
	module    string
	level     Level
	logger    *log.Logger
	isTTY     bool
	useColors bool
}

// NewSimpleLogger creates a new SimpleLogger
func NewSimpleLogger(module string, level Level, useColors bool) *SimpleLogger {
	isTTY := checkTTY()
	return &SimpleLogger{
		module:    module,
		level:     level,
		logger:    log.New(os.Stdout, "", log.LstdFlags),
		isTTY:     isTTY,
		useColors: useColors && isTTY, // Only use colors if enabled and output is a TTY
	}
}

// checkTTY checks if stdout is a terminal
func checkTTY() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// formatMessage formats a log message with level, path, module, and message
// Format: LEVEL  @path [module] message key=value
// The "path" key is treated specially and displayed as @path
func (l *SimpleLogger) formatMessage(level Level, msg string, args ...interface{}) string {
	var pathValue string
	var pairs []string

	// Extract path and build key-value pairs
	if len(args) > 0 {
		for i := 0; i < len(args); i += 2 {
			if i+1 < len(args) {
				key := fmt.Sprintf("%v", args[i])
				value := fmt.Sprintf("%v", args[i+1])

				// Special handling for "path" key
				if key == "path" {
					pathValue = value
				} else {
					pairs = append(pairs, fmt.Sprintf("%s=%s", key, value))
				}
			}
		}
	}

	// Build message with key-value pairs
	message := msg
	if len(pairs) > 0 {
		message = fmt.Sprintf("%s %s", msg, strings.Join(pairs, " "))
	}

	// Format level (left-aligned, 5 characters for consistent formatting)
	levelPart := fmt.Sprintf("%-5s", level.String())
	if l.useColors {
		levelPart = l.colorizeLevel(level, levelPart)
	}

	// Format path if present
	pathPart := ""
	if pathValue != "" {
		pathPart = fmt.Sprintf("@%s ", pathValue)
		if l.useColors {
			pathPart = colorMagenta + pathPart + colorReset
		}
	}

	// Format module name
	modulePart := fmt.Sprintf("[%s]", l.module)
	if l.useColors {
		modulePart = colorCyan + modulePart + colorReset
	}

	return fmt.Sprintf("%s %s%s %s", levelPart, pathPart, modulePart, message)
}

// colorizeLevel applies color to log level
func (l *SimpleLogger) colorizeLevel(level Level, text string) string {
	switch level {
	case LevelDebug:
		return colorGray + text + colorReset
	case LevelInfo:
		return colorGreen + text + colorReset
	case LevelWarn:
		return colorYellow + text + colorReset
	case LevelError:
		return colorRed + text + colorReset
	case LevelFatal:
		return colorRed + colorBold + text + colorReset
	default:
		return text
	}
}

// log is the internal logging method
func (l *SimpleLogger) log(level Level, msg string, args ...interface{}) {
	if level < l.level {
		return
	}

	formatted := l.formatMessage(level, msg, args...)
	l.logger.Println(formatted)

	if level == LevelFatal {
		os.Exit(1)
	}
}

// Debug logs a debug message
func (l *SimpleLogger) Debug(msg string, args ...interface{}) {
	l.log(LevelDebug, msg, args...)
}

// Info logs an informational message
func (l *SimpleLogger) Info(msg string, args ...interface{}) {
	l.log(LevelInfo, msg, args...)
}

// Warn logs a warning message
func (l *SimpleLogger) Warn(msg string, args ...interface{}) {
	l.log(LevelWarn, msg, args...)
}

// Error logs an error message
func (l *SimpleLogger) Error(msg string, args ...interface{}) {
	l.log(LevelError, msg, args...)
}

// Fatal logs a fatal error message and exits
func (l *SimpleLogger) Fatal(msg string, args ...interface{}) {
	l.log(LevelFatal, msg, args...)
}

// WithModule creates a new logger with a hierarchical component name.
// If the current logger already has a component (module), the new component
// is appended with "/" as a separator (e.g., "proxy/middleware/session").
// This allows tracing the origin of log messages through component hierarchy.
func (l *SimpleLogger) WithModule(module string) Logger {
	newModule := module
	if l.module != "" {
		// Append to existing component hierarchy
		newModule = l.module + "/" + module
	}
	return &SimpleLogger{
		module:    newModule,
		level:     l.level,
		logger:    l.logger,
		isTTY:     l.isTTY,
		useColors: l.useColors,
	}
}

// Color codes
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorGray    = "\033[90m"
	colorBold    = "\033[1m"
)
