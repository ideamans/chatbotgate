package email

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileWriter_WriteOTP(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "otp-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "otp.json")
	writer := NewFileWriter(filePath)

	// Test data
	email := "test@example.com"
	token := "test-token-123"
	loginURL := "http://localhost:4180/_auth/email/verify?token=test-token-123"
	expiresAt := time.Now().Add(15 * time.Minute)

	// Write OTP
	err = writer.WriteOTP(email, token, loginURL, expiresAt)
	if err != nil {
		t.Fatalf("WriteOTP() error = %v", err)
	}

	// Read and verify file contents
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read OTP file: %v", err)
	}

	// Parse JSON
	var record OTPRecord
	if err := json.Unmarshal(data[:len(data)-1], &record); err != nil { // Remove trailing newline
		t.Fatalf("Failed to unmarshal OTP record: %v", err)
	}

	// Verify record
	if record.Email != email {
		t.Errorf("Email = %s, want %s", record.Email, email)
	}
	if record.Token != token {
		t.Errorf("Token = %s, want %s", record.Token, token)
	}
	if record.LoginURL != loginURL {
		t.Errorf("LoginURL = %s, want %s", record.LoginURL, loginURL)
	}
	if !record.ExpiresAt.Equal(expiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", record.ExpiresAt, expiresAt)
	}
}

func TestFileWriter_WriteOTP_MultipleRecords(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "otp-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "otp.json")
	writer := NewFileWriter(filePath)

	// Write multiple OTP records
	records := []struct {
		email    string
		token    string
		loginURL string
	}{
		{"user1@example.com", "token-1", "http://localhost:4180/verify?token=token-1"},
		{"user2@example.com", "token-2", "http://localhost:4180/verify?token=token-2"},
		{"user3@example.com", "token-3", "http://localhost:4180/verify?token=token-3"},
	}

	expiresAt := time.Now().Add(15 * time.Minute)
	for _, r := range records {
		if err := writer.WriteOTP(r.email, r.token, r.loginURL, expiresAt); err != nil {
			t.Fatalf("WriteOTP() error = %v", err)
		}
	}

	// Read file contents
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read OTP file: %v", err)
	}

	// Verify JSON Lines format (each line is a separate JSON object)
	lines := 0
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines++
			var record OTPRecord
			if err := json.Unmarshal(data[start:i], &record); err != nil {
				t.Fatalf("Failed to unmarshal line %d: %v", lines, err)
			}
			start = i + 1
		}
	}

	if lines != len(records) {
		t.Errorf("Expected %d lines, got %d", len(records), lines)
	}
}

func TestFileWriter_WriteOTP_FilePermissions(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "otp-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "otp.json")
	writer := NewFileWriter(filePath)

	// Write OTP
	expiresAt := time.Now().Add(15 * time.Minute)
	if err := writer.WriteOTP("test@example.com", "token", "http://localhost/verify", expiresAt); err != nil {
		t.Fatalf("WriteOTP() error = %v", err)
	}

	// Check file permissions
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	mode := info.Mode().Perm()
	expectedMode := os.FileMode(0600)
	if mode != expectedMode {
		t.Errorf("File mode = %o, want %o", mode, expectedMode)
	}
}
