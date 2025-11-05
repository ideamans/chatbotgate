package logging

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{LevelFatal, "FATAL"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("Level.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  Level
	}{
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"info", LevelInfo},
		{"INFO", LevelInfo},
		{"warn", LevelWarn},
		{"WARN", LevelWarn},
		{"warning", LevelWarn},
		{"error", LevelError},
		{"ERROR", LevelError},
		{"fatal", LevelFatal},
		{"FATAL", LevelFatal},
		{"unknown", LevelInfo}, // default
		{"", LevelInfo},        // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ParseLevel(tt.input); got != tt.want {
				t.Errorf("ParseLevel(%s) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSimpleLogger_Log(t *testing.T) {
	var buf bytes.Buffer
	logger := &SimpleLogger{
		module:    "test",
		level:     LevelDebug,
		logger:    log.New(&buf, "", 0),
		isTTY:     false,
		useColors: false,
	}

	tests := []struct {
		name     string
		logFunc  func(string, ...interface{})
		msg      string
		args     []interface{}
		contains []string
	}{
		{
			name:    "debug message",
			logFunc: logger.Debug,
			msg:     "debug message",
			contains: []string{
				"[test]",
				"DEBUG",
				"debug message",
			},
		},
		{
			name:    "info message",
			logFunc: logger.Info,
			msg:     "info message",
			contains: []string{
				"[test]",
				"INFO",
				"info message",
			},
		},
		{
			name:    "warn message",
			logFunc: logger.Warn,
			msg:     "warn message",
			contains: []string{
				"[test]",
				"WARN",
				"warn message",
			},
		},
		{
			name:    "error message",
			logFunc: logger.Error,
			msg:     "error message",
			contains: []string{
				"[test]",
				"ERROR",
				"error message",
			},
		},
		{
			name:    "message with args",
			logFunc: logger.Info,
			msg:     "server started",
			args:    []interface{}{"port", 4180, "host", "localhost"},
			contains: []string{
				"[test]",
				"INFO",
				"server started",
				"port=4180",
				"host=localhost",
			},
		},
		{
			name:    "message with path",
			logFunc: logger.Info,
			msg:     "authentication successful",
			args:    []interface{}{"path", "/_auth/login", "email", "user@example.com"},
			contains: []string{
				"@/_auth/login",
				"[test]",
				"INFO",
				"authentication successful",
				"email=user@example.com",
			},
		},
		{
			name:    "message with path only",
			logFunc: logger.Debug,
			msg:     "handling request",
			args:    []interface{}{"path", "/api/users"},
			contains: []string{
				"@/api/users",
				"[test]",
				"DEBUG",
				"handling request",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc(tt.msg, tt.args...)
			output := buf.String()

			for _, substr := range tt.contains {
				if !strings.Contains(output, substr) {
					t.Errorf("log output missing %q\ngot: %s", substr, output)
				}
			}
		})
	}
}

func TestSimpleLogger_LogLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := &SimpleLogger{
		module:    "test",
		level:     LevelWarn, // Only WARN and above
		logger:    log.New(&buf, "", 0),
		isTTY:     false,
		useColors: false,
	}

	tests := []struct {
		name      string
		logFunc   func(string, ...interface{})
		msg       string
		shouldLog bool
	}{
		{
			name:      "debug should not log",
			logFunc:   logger.Debug,
			msg:       "debug",
			shouldLog: false,
		},
		{
			name:      "info should not log",
			logFunc:   logger.Info,
			msg:       "info",
			shouldLog: false,
		},
		{
			name:      "warn should log",
			logFunc:   logger.Warn,
			msg:       "warn",
			shouldLog: true,
		},
		{
			name:      "error should log",
			logFunc:   logger.Error,
			msg:       "error",
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc(tt.msg)
			output := buf.String()

			if tt.shouldLog && output == "" {
				t.Errorf("expected log output but got none")
			}
			if !tt.shouldLog && output != "" {
				t.Errorf("expected no log output but got: %s", output)
			}
		})
	}
}

func TestSimpleLogger_WithModule(t *testing.T) {
	var buf bytes.Buffer
	logger := &SimpleLogger{
		module:    "main",
		level:     LevelInfo,
		logger:    log.New(&buf, "", 0),
		isTTY:     false,
		useColors: false,
	}

	subLogger := logger.WithModule("submodule")

	subLogger.Info("test message")
	output := buf.String()

	// Test hierarchical component naming: main -> main/submodule
	if !strings.Contains(output, "[main/submodule]") {
		t.Errorf("expected [main/submodule] in output (hierarchical component), got: %s", output)
	}
}

func TestSimpleLogger_WithModule_Hierarchy(t *testing.T) {
	var buf bytes.Buffer
	logger := &SimpleLogger{
		module:    "main",
		level:     LevelInfo,
		logger:    log.New(&buf, "", 0),
		isTTY:     false,
		useColors: false,
	}

	// Create nested hierarchy: main -> main/manager -> main/manager/middleware
	managerLogger := logger.WithModule("manager")
	middlewareLogger := managerLogger.WithModule("middleware")

	middlewareLogger.Info("test message")
	output := buf.String()

	// Test multi-level hierarchical component naming
	if !strings.Contains(output, "[main/manager/middleware]") {
		t.Errorf("expected [main/manager/middleware] in output (multi-level hierarchy), got: %s", output)
	}
}

func TestSimpleLogger_WithModule_EmptyParent(t *testing.T) {
	var buf bytes.Buffer
	logger := &SimpleLogger{
		module:    "", // Empty parent component
		level:     LevelInfo,
		logger:    log.New(&buf, "", 0),
		isTTY:     false,
		useColors: false,
	}

	subLogger := logger.WithModule("component")

	subLogger.Info("test message")
	output := buf.String()

	// When parent is empty, should just use the new component name
	if !strings.Contains(output, "[component]") {
		t.Errorf("expected [component] in output (no parent), got: %s", output)
	}
	// Should not have a leading slash
	if strings.Contains(output, "[/component]") {
		t.Errorf("unexpected leading slash in output: %s", output)
	}
}

func TestCheckTTY(t *testing.T) {
	// This test just ensures the function doesn't panic
	// We can't reliably test the actual TTY detection in automated tests
	isTTY := checkTTY()
	t.Logf("checkTTY() returned %v", isTTY)
}

func TestSimpleLogger_Colors(t *testing.T) {
	var buf bytes.Buffer

	// Test with colors enabled (simulating TTY)
	logger := &SimpleLogger{
		module:    "test",
		level:     LevelDebug,
		logger:    log.New(&buf, "", 0),
		isTTY:     true,
		useColors: true,
	}

	logger.Info("colored message")
	output := buf.String()

	// Should contain color codes when colors are enabled
	if !strings.Contains(output, "\033[") {
		t.Error("expected color codes in output when colors are enabled")
	}

	// Test with colors disabled
	buf.Reset()
	logger.useColors = false
	logger.Info("non-colored message")
	output = buf.String()

	// Should not contain color codes when colors are disabled
	if strings.Contains(output, "\033[") {
		t.Error("unexpected color codes in output when colors are disabled")
	}
}

// TestFatal is skipped because it calls os.Exit
// In a real-world scenario, you would test this with a subprocess
func TestSimpleLogger_Fatal_Skip(t *testing.T) {
	t.Skip("Skipping fatal test as it calls os.Exit")

	// Example of how you might test this in production:
	// if os.Getenv("TEST_FATAL") == "1" {
	//     logger := NewSimpleLogger("test", LevelDebug, false)
	//     logger.Fatal("fatal error")
	//     return
	// }
	// cmd := exec.Command(os.Args[0], "-test.run=TestSimpleLogger_Fatal")
	// cmd.Env = append(os.Environ(), "TEST_FATAL=1")
	// err := cmd.Run()
	// if e, ok := err.(*exec.ExitError); ok && !e.Success() {
	//     return // Expected
	// }
	// t.Fatalf("process ran with err %v, want exit status 1", err)
}
