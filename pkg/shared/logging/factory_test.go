package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewLoggerWithFile_NoFileConfig(t *testing.T) {
	logger, err := NewLoggerWithFile("test", LevelInfo, false, nil)
	if err != nil {
		t.Fatalf("Expected no error with nil config, got: %v", err)
	}
	if logger == nil {
		t.Fatal("Expected logger to be created")
	}
}

func TestNewLoggerWithFile_EmptyPath(t *testing.T) {
	config := &FileRotationConfig{
		Path: "",
	}
	logger, err := NewLoggerWithFile("test", LevelInfo, false, config)
	if err != nil {
		t.Fatalf("Expected no error with empty path, got: %v", err)
	}
	if logger == nil {
		t.Fatal("Expected logger to be created")
	}
}

func TestNewLoggerWithFile_WithFile(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	config := &FileRotationConfig{
		Path:       logPath,
		MaxSizeMB:  1, // 1MB
		MaxBackups: 2,
		MaxAge:     7,
		Compress:   false,
	}

	logger, err := NewLoggerWithFile("test", LevelInfo, false, config)
	if err != nil {
		t.Fatalf("Failed to create logger with file: %v", err)
	}
	if logger == nil {
		t.Fatal("Expected logger to be created")
	}

	// Write log messages
	logger.Info("Test message 1")
	logger.Warn("Test message 2", "key", "value")

	// Give it a moment to write
	time.Sleep(100 * time.Millisecond)

	// Check if log file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatalf("Log file was not created at %s", logPath)
	}

	// Read log file and verify content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Test message 1") {
		t.Error("Log file does not contain first message")
	}
	if !strings.Contains(contentStr, "Test message 2") {
		t.Error("Log file does not contain second message")
	}
	if !strings.Contains(contentStr, "key=value") {
		t.Error("Log file does not contain key-value pair")
	}
}

func TestNewLoggerWithFile_DefaultValues(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	// Config with only path set (should use defaults)
	config := &FileRotationConfig{
		Path: logPath,
	}

	logger, err := NewLoggerWithFile("test", LevelInfo, false, config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	if logger == nil {
		t.Fatal("Expected logger to be created")
	}

	// Write a log message
	logger.Info("Default config test")
	time.Sleep(100 * time.Millisecond)

	// Verify file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatalf("Log file was not created with default config")
	}
}

func TestNewLoggerWithFile_MultipleModules(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	config := &FileRotationConfig{
		Path: logPath,
	}

	logger, err := NewLoggerWithFile("main", LevelInfo, false, config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create sub-module loggers
	authLogger := logger.WithModule("auth")
	sessionLogger := authLogger.WithModule("session")

	// Write from different modules
	logger.Info("Main module message")
	authLogger.Info("Auth module message")
	sessionLogger.Info("Session module message")

	time.Sleep(100 * time.Millisecond)

	// Read and verify
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "[main]") {
		t.Error("Missing main module marker")
	}
	if !strings.Contains(contentStr, "[main/auth]") {
		t.Error("Missing auth module marker")
	}
	if !strings.Contains(contentStr, "[main/auth/session]") {
		t.Error("Missing session module marker")
	}
}

func TestNewLoggerWithFile_NoColorsInFile(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	config := &FileRotationConfig{
		Path: logPath,
	}

	// Create logger with colors enabled (should be ignored for file output)
	logger, err := NewLoggerWithFile("test", LevelInfo, true, config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Write various log levels
	logger.Info("Info message")
	logger.Warn("Warning message")
	logger.Error("Error message")

	time.Sleep(100 * time.Millisecond)

	// Read log file
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)

	// Check that no ANSI color codes are present in the file
	// ANSI codes start with \033[ or \x1b[
	if strings.Contains(contentStr, "\033[") || strings.Contains(contentStr, "\x1b[") {
		t.Error("Log file contains ANSI color codes, but colors should be disabled for file output")
	}

	// Verify messages are still present
	if !strings.Contains(contentStr, "Info message") {
		t.Error("Info message not found in log file")
	}
	if !strings.Contains(contentStr, "Warning message") {
		t.Error("Warning message not found in log file")
	}
	if !strings.Contains(contentStr, "Error message") {
		t.Error("Error message not found in log file")
	}
}
