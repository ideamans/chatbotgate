package forwarding

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
)

var (
	// ErrInvalidCiphertext is returned when ciphertext is invalid or corrupted
	ErrInvalidCiphertext = errors.New("invalid ciphertext")

	// ErrDecryptionFailed is returned when decryption fails
	ErrDecryptionFailed = errors.New("decryption failed")
)

// Encryptor handles encryption and decryption of user data
type Encryptor struct {
	key []byte
}

// NewEncryptor creates a new Encryptor with the given key
// The key is hashed with SHA-256 to ensure it's exactly 32 bytes for AES-256
func NewEncryptor(key string) *Encryptor {
	hash := sha256.Sum256([]byte(key))
	return &Encryptor{
		key: hash[:],
	}
}

// Encrypt encrypts the plaintext using AES-256-GCM and returns a base64-encoded string
// The encrypted data format is: [nonce (12 bytes)][ciphertext]
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	// Create AES cipher
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	// Create GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Generate random nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt and authenticate
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode to base64
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts the base64-encoded ciphertext using AES-256-GCM
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", ErrInvalidCiphertext
	}

	// Create AES cipher
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	// Create GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Check minimum length (nonce + at least some data)
	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", ErrInvalidCiphertext
	}

	// Extract nonce and encrypted data
	nonce := data[:nonceSize]
	encryptedData := data[nonceSize:]

	// Decrypt and verify
	plaintext, err := aesGCM.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}

// EncryptMap encrypts a map of key-value pairs and returns a single encrypted string
// The map is serialized as JSON before encryption
func (e *Encryptor) EncryptMap(data map[string]string) (string, error) {
	// Serialize map to JSON
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return e.Encrypt(string(jsonBytes))
}

// DecryptMap decrypts an encrypted string back to a map of key-value pairs
func (e *Encryptor) DecryptMap(ciphertext string) (map[string]string, error) {
	// Decrypt
	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return nil, err
	}

	// Deserialize from JSON
	var result map[string]string
	if err := json.Unmarshal([]byte(plaintext), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// serializeMap converts a map to a URL query string format
func serializeMap(data map[string]string) string {
	if len(data) == 0 {
		return ""
	}

	result := ""
	first := true
	for key, value := range data {
		if !first {
			result += "&"
		}
		result += key + "=" + value
		first = false
	}
	return result
}

// deserializeMap converts a URL query string format back to a map
func deserializeMap(serialized string) map[string]string {
	result := make(map[string]string)
	if serialized == "" {
		return result
	}

	pairs := splitString(serialized, '&')
	for _, pair := range pairs {
		kv := splitString(pair, '=')
		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		}
	}
	return result
}

// splitString splits a string by a delimiter
func splitString(s string, delim rune) []string {
	var result []string
	var current string

	for _, c := range s {
		if c == delim {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}
