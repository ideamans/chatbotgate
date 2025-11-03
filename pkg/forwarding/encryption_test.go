package forwarding

import (
	"strings"
	"testing"
)

func TestEncryptor_EncryptDecrypt(t *testing.T) {
	key := "this-is-a-32-character-encryption-key"
	encryptor := NewEncryptor(key)

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "simple string",
			plaintext: "hello world",
		},
		{
			name:      "empty string",
			plaintext: "",
		},
		{
			name:      "long string",
			plaintext: strings.Repeat("abcdefghijklmnopqrstuvwxyz", 100),
		},
		{
			name:      "multibyte characters (Japanese)",
			plaintext: "こんにちは世界",
		},
		{
			name:      "email",
			plaintext: "user@example.com",
		},
		{
			name:      "username",
			plaintext: "john_doe_123",
		},
		{
			name:      "special characters",
			plaintext: "!@#$%^&*()_+-=[]{}|;':\",./<>?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := encryptor.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Check that ciphertext is not empty
			if ciphertext == "" && tt.plaintext != "" {
				t.Error("Encrypt() returned empty ciphertext for non-empty plaintext")
			}

			// Check that ciphertext is different from plaintext
			if ciphertext == tt.plaintext && tt.plaintext != "" {
				t.Error("Encrypt() returned plaintext as ciphertext")
			}

			// Decrypt
			decrypted, err := encryptor.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			// Check that decrypted matches original
			if decrypted != tt.plaintext {
				t.Errorf("Decrypt() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptor_EncryptRandomness(t *testing.T) {
	key := "this-is-a-32-character-encryption-key"
	encryptor := NewEncryptor(key)
	plaintext := "test data"

	// Encrypt the same plaintext multiple times
	ciphertext1, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	ciphertext2, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Ciphertexts should be different due to random nonce
	if ciphertext1 == ciphertext2 {
		t.Error("Encrypt() produced identical ciphertexts for same plaintext (nonce not random)")
	}

	// But both should decrypt to the same plaintext
	decrypted1, _ := encryptor.Decrypt(ciphertext1)
	decrypted2, _ := encryptor.Decrypt(ciphertext2)

	if decrypted1 != plaintext || decrypted2 != plaintext {
		t.Error("Decrypt() failed to recover original plaintext")
	}
}

func TestEncryptor_DecryptInvalid(t *testing.T) {
	key := "this-is-a-32-character-encryption-key"
	encryptor := NewEncryptor(key)

	tests := []struct {
		name       string
		ciphertext string
		wantErr    error
	}{
		{
			name:       "invalid base64",
			ciphertext: "not-valid-base64!!!",
			wantErr:    ErrInvalidCiphertext,
		},
		{
			name:       "too short",
			ciphertext: "YWJj", // "abc" in base64, too short for nonce
			wantErr:    ErrInvalidCiphertext,
		},
		{
			name:       "corrupted ciphertext",
			ciphertext: "AAAAAAAAAAAAAAAAAAAAAA==", // Valid base64 but invalid ciphertext
			wantErr:    ErrDecryptionFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := encryptor.Decrypt(tt.ciphertext)
			if err != tt.wantErr {
				t.Errorf("Decrypt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncryptor_DifferentKeys(t *testing.T) {
	key1 := "this-is-a-32-character-encryption-key"
	key2 := "different-32-character-encryption-key"

	encryptor1 := NewEncryptor(key1)
	encryptor2 := NewEncryptor(key2)

	plaintext := "secret message"

	// Encrypt with key1
	ciphertext, err := encryptor1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Try to decrypt with key2 (should fail)
	_, err = encryptor2.Decrypt(ciphertext)
	if err != ErrDecryptionFailed {
		t.Errorf("Decrypt() with wrong key should fail, got error = %v", err)
	}
}

func TestEncryptor_EncryptDecryptMap(t *testing.T) {
	key := "this-is-a-32-character-encryption-key"
	encryptor := NewEncryptor(key)

	tests := []struct {
		name string
		data map[string]string
	}{
		{
			name: "single field",
			data: map[string]string{
				"username": "john_doe",
			},
		},
		{
			name: "multiple fields",
			data: map[string]string{
				"username": "john_doe",
				"email":    "john@example.com",
			},
		},
		{
			name: "empty map",
			data: map[string]string{},
		},
		{
			name: "special characters in values",
			data: map[string]string{
				"username": "user@domain",
				"email":    "test+tag@example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := encryptor.EncryptMap(tt.data)
			if err != nil {
				t.Fatalf("EncryptMap() error = %v", err)
			}

			// Decrypt
			decrypted, err := encryptor.DecryptMap(ciphertext)
			if err != nil {
				t.Fatalf("DecryptMap() error = %v", err)
			}

			// Compare maps
			if len(decrypted) != len(tt.data) {
				t.Errorf("DecryptMap() length = %v, want %v", len(decrypted), len(tt.data))
			}

			for key, value := range tt.data {
				if decrypted[key] != value {
					t.Errorf("DecryptMap()[%s] = %v, want %v", key, decrypted[key], value)
				}
			}
		})
	}
}

func TestSerializeDeserializeMap(t *testing.T) {
	tests := []struct {
		name string
		data map[string]string
	}{
		{
			name: "single field",
			data: map[string]string{"username": "john"},
		},
		{
			name: "multiple fields",
			data: map[string]string{"username": "john", "email": "john@example.com"},
		},
		{
			name: "empty map",
			data: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serialized := serializeMap(tt.data)
			deserialized := deserializeMap(serialized)

			if len(deserialized) != len(tt.data) {
				t.Errorf("deserializeMap() length = %v, want %v", len(deserialized), len(tt.data))
			}

			for key, value := range tt.data {
				if deserialized[key] != value {
					t.Errorf("deserializeMap()[%s] = %v, want %v", key, deserialized[key], value)
				}
			}
		})
	}
}

func TestSplitString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		delim rune
		want  []string
	}{
		{
			name:  "simple split",
			input: "a&b&c",
			delim: '&',
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "split with equals",
			input: "key=value",
			delim: '=',
			want:  []string{"key", "value"},
		},
		{
			name:  "empty string",
			input: "",
			delim: '&',
			want:  []string{},
		},
		{
			name:  "no delimiter",
			input: "abc",
			delim: '&',
			want:  []string{"abc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitString(tt.input, tt.delim)
			if len(got) != len(tt.want) {
				t.Errorf("splitString() length = %v, want %v", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitString()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
