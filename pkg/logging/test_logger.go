package logging

import "testing"

// TestLogger is a logger for testing that suppresses output
type TestLogger struct {
	module string
	t      *testing.T
}

// NewTestLogger creates a new test logger that suppresses output
// If you need to see logs during tests, use NewTestLoggerVerbose instead
func NewTestLogger() *TestLogger {
	return &TestLogger{
		module: "test",
	}
}

// NewTestLoggerVerbose creates a test logger that outputs to testing.T
func NewTestLoggerVerbose(t *testing.T) *TestLogger {
	return &TestLogger{
		module: "test",
		t:      t,
	}
}

// Debug logs a debug message
func (l *TestLogger) Debug(msg string, args ...interface{}) {
	if l.t != nil {
		l.t.Logf("[%s] DEBUG: %s %v", l.module, msg, args)
	}
}

// Info logs an informational message
func (l *TestLogger) Info(msg string, args ...interface{}) {
	if l.t != nil {
		l.t.Logf("[%s] INFO: %s %v", l.module, msg, args)
	}
}

// Warn logs a warning message
func (l *TestLogger) Warn(msg string, args ...interface{}) {
	if l.t != nil {
		l.t.Logf("[%s] WARN: %s %v", l.module, msg, args)
	}
}

// Error logs an error message
func (l *TestLogger) Error(msg string, args ...interface{}) {
	if l.t != nil {
		l.t.Logf("[%s] ERROR: %s %v", l.module, msg, args)
	}
}

// Fatal logs a fatal error message (but doesn't exit in tests)
func (l *TestLogger) Fatal(msg string, args ...interface{}) {
	if l.t != nil {
		l.t.Fatalf("[%s] FATAL: %s %v", l.module, msg, args)
	}
}

// WithModule creates a new logger with a hierarchical component name.
// If the current logger already has a component (module), the new component
// is appended with "/" as a separator (e.g., "proxy/middleware/session").
func (l *TestLogger) WithModule(module string) Logger {
	newModule := module
	if l.module != "" {
		// Append to existing component hierarchy
		newModule = l.module + "/" + module
	}
	return &TestLogger{
		module: newModule,
		t:      l.t,
	}
}
