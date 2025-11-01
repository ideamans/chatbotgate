package email

import (
	"encoding/json"
	"fmt"
	"os"
	"syscall"
	"time"
)

// OTPRecord represents a single OTP record for E2E testing
type OTPRecord struct {
	Email     string    `json:"email"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	LoginURL  string    `json:"login_url"`
}

// FileWriter handles writing OTP records to a file
type FileWriter struct {
	filePath string
}

// NewFileWriter creates a new FileWriter
func NewFileWriter(filePath string) *FileWriter {
	return &FileWriter{
		filePath: filePath,
	}
}

// WriteOTP appends an OTP record to the file in JSON Lines format
func (w *FileWriter) WriteOTP(email, token, loginURL string, expiresAt time.Time) error {
	record := OTPRecord{
		Email:     email,
		Token:     token,
		ExpiresAt: expiresAt,
		LoginURL:  loginURL,
	}

	// Serialize to JSON
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal OTP record: %w", err)
	}

	// Append newline for JSON Lines format
	data = append(data, '\n')

	// Open file with O_APPEND flag for atomic append
	// Create file if it doesn't exist with 0600 permissions
	file, err := os.OpenFile(w.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open OTP file: %w", err)
	}
	defer file.Close()

	// Acquire exclusive lock
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to lock OTP file: %w", err)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	// Write data
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write OTP record: %w", err)
	}

	return nil
}
