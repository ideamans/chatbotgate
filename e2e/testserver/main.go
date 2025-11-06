package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/ideamans/chatbotgate/pkg/middleware/forwarding"
)

// UserInfoResponse represents the response with user information
type UserInfoResponse struct {
	QueryString *UserData  `json:"querystring,omitempty"`
	Header      *UserData  `json:"header,omitempty"`
	RawHeaders  RawHeaders `json:"raw_headers,omitempty"`
}

// UserData contains user information
type UserData struct {
	Username  string `json:"username,omitempty"`
	Email     string `json:"email,omitempty"`
	Encrypted bool   `json:"encrypted"`
}

// RawHeaders contains raw header values
type RawHeaders struct {
	ForwardedUser  string `json:"X-ChatbotGate-User,omitempty"`
	ForwardedEmail string `json:"X-ChatbotGate-Email,omitempty"`
}

var encryptionKey string

func main() {
	port := flag.Int("port", 8083, "Port to listen on")
	key := flag.String("key", "", "Encryption key for decrypting user data")
	flag.Parse()

	if *key == "" {
		log.Fatal("Encryption key is required (use -key flag)")
	}
	encryptionKey = *key

	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/health", handleHealth)

	// Passthrough test endpoints
	http.HandleFunc("/embed.js", handleEmbedJS)
	http.HandleFunc("/public/data.json", handlePublicData)
	http.HandleFunc("/static/image.png", handleStaticImage)
	http.HandleFunc("/api/public/info", handlePublicAPI)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Test backend server starting on %s", addr)
	log.Printf("Encryption key: %s", encryptionKey)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	response := UserInfoResponse{
		RawHeaders: RawHeaders{
			ForwardedUser:  r.Header.Get("X-ChatbotGate-User"),
			ForwardedEmail: r.Header.Get("X-ChatbotGate-Email"),
		},
	}

	// Extract querystring data from individual parameters
	// Try both old names (chatbotgate.user/email) and new names (username/email)
	usernameParam := r.URL.Query().Get("username")
	if usernameParam == "" {
		usernameParam = r.URL.Query().Get("chatbotgate.user")
	}

	emailParam := r.URL.Query().Get("email")
	if emailParam == "" {
		emailParam = r.URL.Query().Get("chatbotgate.email")
	}

	if usernameParam != "" || emailParam != "" {
		// Try to decrypt individual fields
		userData := &UserData{}
		encrypted := false

		if usernameParam != "" {
			// Try decryption first
			if decrypted := decryptField(usernameParam); decrypted != "" {
				userData.Username = decrypted
				encrypted = true
			} else {
				// Fall back to plain text
				userData.Username = usernameParam
			}
		}

		if emailParam != "" {
			// Try decryption first
			if decrypted := decryptField(emailParam); decrypted != "" {
				userData.Email = decrypted
				encrypted = true
			} else {
				// Fall back to plain text
				userData.Email = emailParam
			}
		}

		userData.Encrypted = encrypted
		response.QueryString = userData
	}

	// Extract header data from individual headers
	forwardedUser := r.Header.Get("X-ChatbotGate-User")
	forwardedEmail := r.Header.Get("X-ChatbotGate-Email")

	if forwardedUser != "" || forwardedEmail != "" {
		// Try to decrypt individual fields
		userData := &UserData{}
		encrypted := false

		if forwardedUser != "" {
			// Try decryption first
			if decrypted := decryptField(forwardedUser); decrypted != "" {
				userData.Username = decrypted
				encrypted = true
			} else {
				// Fall back to plain text
				userData.Username = forwardedUser
			}
		}

		if forwardedEmail != "" {
			// Try decryption first
			if decrypted := decryptField(forwardedEmail); decrypted != "" {
				userData.Email = decrypted
				encrypted = true
			} else {
				// Fall back to plain text
				userData.Email = forwardedEmail
			}
		}

		userData.Encrypted = encrypted
		response.Header = userData
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// decryptField attempts to decrypt a single field value
// Returns decrypted string on success, empty string on failure
func decryptField(encrypted string) string {
	encryptor := forwarding.NewEncryptor(encryptionKey)

	// The filter chain auto-applies base64 encoding when encrypt filter outputs binary type
	// So the data is double base64-encoded: we need to decode once first
	outerDecoded, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		// Not double-encoded, try decrypting directly
		decrypted, err := encryptor.Decrypt(encrypted)
		if err != nil {
			return ""
		}
		return decrypted
	}

	// Now decrypt (which will decode the inner base64 internally)
	decrypted, err := encryptor.Decrypt(string(outerDecoded))
	if err != nil {
		// Not encrypted or decryption failed
		return ""
	}
	return decrypted
}

// Passthrough test handlers
// These handlers should be accessible without authentication when passthrough is configured

func handleEmbedJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("// Embed widget script\nconsole.log('ChatbotGate embed widget loaded');"))
}

func handlePublicData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "public data",
		"status":  "ok",
	})
}

func handleStaticImage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	// Return a 1x1 transparent PNG
	png := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	}
	_, _ = w.Write(png)
}

func handlePublicAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"api":           "public",
		"version":       "1.0",
		"authenticated": false,
	})
}
