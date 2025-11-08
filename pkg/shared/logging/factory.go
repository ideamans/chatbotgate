package logging

import (
	"io"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// FileRotationConfig contains file logging rotation settings
type FileRotationConfig struct {
	Path       string // Log file path (required)
	MaxSizeMB  int    // Maximum size in megabytes before rotation (default: 100)
	MaxBackups int    // Maximum number of old log files to retain (default: 3)
	MaxAge     int    // Maximum number of days to retain old log files (default: 28)
	Compress   bool   // Whether to compress rotated log files (default: false)
}

// NewLoggerWithFile creates a logger that writes to both console and file with rotation
// When file logging is enabled, colors are always disabled for file output to avoid ANSI escape codes in log files
func NewLoggerWithFile(module string, level Level, useColors bool, fileConfig *FileRotationConfig) (*SimpleLogger, error) {
	// If no file config, return console-only logger
	if fileConfig == nil || fileConfig.Path == "" {
		return NewSimpleLogger(module, level, useColors), nil
	}

	// Set defaults for rotation settings
	maxSizeMB := fileConfig.MaxSizeMB
	if maxSizeMB == 0 {
		maxSizeMB = 100 // 100MB default
	}

	maxBackups := fileConfig.MaxBackups
	if maxBackups == 0 {
		maxBackups = 3 // Keep 3 old files by default
	}

	maxAge := fileConfig.MaxAge
	if maxAge == 0 {
		maxAge = 28 // Keep files for 28 days by default
	}

	// Create lumberjack logger for file rotation
	fileWriter := &lumberjack.Logger{
		Filename:   fileConfig.Path,
		MaxSize:    maxSizeMB,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   fileConfig.Compress,
	}

	// Create multi-writer for both console and file
	// File output always has colors disabled to avoid ANSI escape codes
	multiWriter := io.MultiWriter(os.Stdout, fileWriter)

	// Disable colors when writing to file to avoid ANSI escape codes in log files
	return NewSimpleLoggerWithWriter(module, level, false, multiWriter), nil
}
