package forwarding

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"testing"
)

func TestEncryptFilter(t *testing.T) {
	encryptor := NewEncryptor("this-is-a-32-character-encryption-key-12345")
	filter := NewEncryptFilter(encryptor)

	input := &FilterOutput{
		Data: []byte("hello world"),
		Type: DataTypeString,
	}

	output, err := filter.Apply(input)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if output.Type != DataTypeBinary {
		t.Errorf("OutputType = %v, want %v", output.Type, DataTypeBinary)
	}

	// Verify we can decrypt
	decrypted, err := encryptor.Decrypt(string(output.Data))
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if decrypted != "hello world" {
		t.Errorf("Decrypted = %v, want %v", decrypted, "hello world")
	}
}

func TestZipFilter(t *testing.T) {
	filter := NewZipFilter()

	input := &FilterOutput{
		Data: []byte("hello world hello world hello world"),
		Type: DataTypeString,
	}

	output, err := filter.Apply(input)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if output.Type != DataTypeBinary {
		t.Errorf("OutputType = %v, want %v", output.Type, DataTypeBinary)
	}

	// Verify we can decompress
	reader, err := gzip.NewReader(bytes.NewReader(output.Data))
	if err != nil {
		t.Fatalf("gzip.NewReader() error = %v", err)
	}
	defer func() { _ = reader.Close() }()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(decompressed) != "hello world hello world hello world" {
		t.Errorf("Decompressed = %v, want %v", string(decompressed), "hello world hello world hello world")
	}
}

func TestBase64Filter(t *testing.T) {
	filter := NewBase64Filter()

	input := &FilterOutput{
		Data: []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f}, // "Hello"
		Type: DataTypeBinary,
	}

	output, err := filter.Apply(input)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if output.Type != DataTypeString {
		t.Errorf("OutputType = %v, want %v", output.Type, DataTypeString)
	}

	expected := base64.StdEncoding.EncodeToString([]byte{0x48, 0x65, 0x6c, 0x6c, 0x6f})
	if string(output.Data) != expected {
		t.Errorf("Output = %v, want %v", string(output.Data), expected)
	}
}

func TestFilterChain_EncryptOnly(t *testing.T) {
	encryptor := NewEncryptor("this-is-a-32-character-encryption-key-12345")
	chain, err := NewFilterChain([]string{"encrypt"}, encryptor)
	if err != nil {
		t.Fatalf("NewFilterChain() error = %v", err)
	}

	result, err := chain.Apply("hello world")
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	// Result should be base64-encoded encrypted data (auto base64)
	decoded, err := base64.StdEncoding.DecodeString(result)
	if err != nil {
		t.Fatalf("base64 decode error = %v", err)
	}

	// Decrypt
	decrypted, err := encryptor.Decrypt(string(decoded))
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if decrypted != "hello world" {
		t.Errorf("Decrypted = %v, want %v", decrypted, "hello world")
	}
}

func TestFilterChain_EncryptThenZip(t *testing.T) {
	encryptor := NewEncryptor("this-is-a-32-character-encryption-key-12345")
	chain, err := NewFilterChain([]string{"encrypt", "zip"}, encryptor)
	if err != nil {
		t.Fatalf("NewFilterChain() error = %v", err)
	}

	result, err := chain.Apply("hello world")
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	// Result should be base64-encoded (auto)
	// Step 1: Decode base64
	compressed, err := base64.StdEncoding.DecodeString(result)
	if err != nil {
		t.Fatalf("base64 decode error = %v", err)
	}

	// Step 2: Decompress gzip
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("gzip.NewReader() error = %v", err)
	}
	defer func() { _ = reader.Close() }()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	// Step 3: Decrypt (decompressed is encrypted base64 string)
	decrypted, err := encryptor.Decrypt(string(decompressed))
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if decrypted != "hello world" {
		t.Errorf("Decrypted = %v, want %v", decrypted, "hello world")
	}
}

func TestFilterChain_ZipOnly(t *testing.T) {
	chain, err := NewFilterChain([]string{"zip"}, nil)
	if err != nil {
		t.Fatalf("NewFilterChain() error = %v", err)
	}

	result, err := chain.Apply("hello world hello world hello world")
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	// Result should be base64-encoded compressed data
	compressed, err := base64.StdEncoding.DecodeString(result)
	if err != nil {
		t.Fatalf("base64 decode error = %v", err)
	}

	// Decompress
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("gzip.NewReader() error = %v", err)
	}
	defer func() { _ = reader.Close() }()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(decompressed) != "hello world hello world hello world" {
		t.Errorf("Decompressed = %v, want %v", string(decompressed), "hello world hello world hello world")
	}
}

func TestFilterChain_NoFilters(t *testing.T) {
	chain, err := NewFilterChain([]string{}, nil)
	if err != nil {
		t.Fatalf("NewFilterChain() error = %v", err)
	}

	result, err := chain.Apply("hello world")
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if result != "hello world" {
		t.Errorf("Result = %v, want %v", result, "hello world")
	}
}

func TestFilterChain_EncryptWithoutEncryptor(t *testing.T) {
	_, err := NewFilterChain([]string{"encrypt"}, nil)
	if err == nil {
		t.Error("NewFilterChain() expected error for encrypt without encryptor, got nil")
	}
}

func TestFilterChain_UnknownFilter(t *testing.T) {
	_, err := NewFilterChain([]string{"unknown"}, nil)
	if err == nil {
		t.Error("NewFilterChain() expected error for unknown filter, got nil")
	}
}
