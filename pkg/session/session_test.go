package session

import (
	"testing"
	"time"
)

func TestSession_IsValid(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		session *Session
		want    bool
	}{
		{
			name: "valid session",
			session: &Session{
				ID:            "test-id",
				Email:         "user@example.com",
				Provider:      "google",
				CreatedAt:     now,
				ExpiresAt:     now.Add(1 * time.Hour),
				Authenticated: true,
			},
			want: true,
		},
		{
			name: "expired session",
			session: &Session{
				ID:            "test-id",
				Email:         "user@example.com",
				Provider:      "google",
				CreatedAt:     now.Add(-2 * time.Hour),
				ExpiresAt:     now.Add(-1 * time.Hour),
				Authenticated: true,
			},
			want: false,
		},
		{
			name: "not authenticated",
			session: &Session{
				ID:            "test-id",
				Email:         "user@example.com",
				Provider:      "google",
				CreatedAt:     now,
				ExpiresAt:     now.Add(1 * time.Hour),
				Authenticated: false,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.session.IsValid(); got != tt.want {
				t.Errorf("Session.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
