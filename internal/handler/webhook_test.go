package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"deploymate/internal/preview"
)

func TestVerifySignature(t *testing.T) {
	tests := []struct {
		name      string
		payload   []byte
		secret    string
		signature string
		want      bool
	}{
		{
			name:      "valid signature",
			payload:   []byte(`{"action":"opened"}`),
			secret:    "my-secret",
			signature: "",
			want:      true,
		},
		{
			name:      "empty signature",
			payload:   []byte(`{"action":"opened"}`),
			secret:    "my-secret",
			signature: "",
			want:      false,
		},
		{
			name:      "invalid signature",
			payload:   []byte(`{"action":"opened"}`),
			secret:    "my-secret",
			signature: "sha256=invalid",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mac := hmac.New(sha256.New, []byte(tt.secret))
			mac.Write(tt.payload)
			expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

			if tt.signature == "" && tt.want {
				tt.signature = expected
			}

			got := verifySignature(tt.payload, tt.signature, tt.secret)
			if got != tt.want {
				t.Errorf("verifySignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandleGitHubWebhook_PROpened(t *testing.T) {
	mgr := preview.NewManager()
	handler := NewWebhookHandler("", mgr)

	event := GitHubWebhookEvent{
		Action: "opened",
		Number: 42,
	}
	event.PR.Head.Ref = "feature/new-api"
	event.PR.Head.SHA = "abc123"
	event.PR.Base.Repo.CloneURL = "https://github.com/example/repo.git"

	body, _ := json.Marshal(event)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "pull_request")
	w := httptest.NewRecorder()

	handler.HandleGitHubWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestHandleGitHubWebhook_InvalidMethod(t *testing.T) {
	mgr := preview.NewManager()
	handler := NewWebhookHandler("", mgr)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/github", nil)
	w := httptest.NewRecorder()

	handler.HandleGitHubWebhook(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", w.Code)
	}
}

func TestHandleGitHubWebhook_InvalidJSON(t *testing.T) {
	mgr := preview.NewManager()
	handler := NewWebhookHandler("", mgr)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()

	handler.HandleGitHubWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestHandleGitHubWebhook_BadSignature(t *testing.T) {
	mgr := preview.NewManager()
	handler := NewWebhookHandler("my-secret", mgr)

	event := GitHubWebhookEvent{Action: "opened", Number: 42}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", "sha256=bad")
	w := httptest.NewRecorder()

	handler.HandleGitHubWebhook(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}
