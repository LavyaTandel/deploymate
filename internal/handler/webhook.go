package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"

	"deploymate/internal/preview"
)

// WebhookHandler handles GitHub webhook events
type WebhookHandler struct {
	secret     string
	previewMgr *preview.Manager
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(secret string, previewMgr *preview.Manager) *WebhookHandler {
	return &WebhookHandler{
		secret:     secret,
		previewMgr: previewMgr,
	}
}

// GitHubWebhookEvent represents a GitHub pull_request webhook event
type GitHubWebhookEvent struct {
	Action string `json:"action"`
	Number int    `json:"number"`
	PR     struct {
		Head struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"head"`
		Base struct {
			Ref  string `json:"ref"`
			Repo struct {
				CloneURL string `json:"clone_url"`
			} `json:"repo"`
		} `json:"base"`
	} `json:"pull_request"`
}

// HandleGitHubWebhook processes GitHub webhook events
func (h *WebhookHandler) HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verify webhook signature if secret is configured
	if h.secret != "" {
		sig := r.Header.Get("X-Hub-Signature-256")
		if !verifySignature(body, sig, h.secret) {
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	eventType := r.Header.Get("X-GitHub-Event")

	// Parse event
	var event GitHubWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Handle PR events asynchronously
	if eventType == "pull_request" {
		go h.handlePREvent(r.Context(), event)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *WebhookHandler) handlePREvent(ctx context.Context, event GitHubWebhookEvent) {
	switch event.Action {
	case "opened", "synchronize":
		req := preview.PreviewRequest{
			RepoURL:   event.PR.Base.Repo.CloneURL,
			PRNumber:  event.Number,
			Branch:    event.PR.Head.Ref,
			Service:   extractService(event.PR.Head.Ref),
			Image:     "",
			CommitSHA: event.PR.Head.SHA,
		}
		_, err := h.previewMgr.CreatePreview(ctx, req)
		if err != nil {
			log.Printf("Failed to create preview for PR #%d: %v", event.Number, err)
		}

	case "closed":
		previewID := extractService(event.PR.Head.Ref) + "-pr-" + strconv.Itoa(event.Number)
		if err := h.previewMgr.DestroyPreview(ctx, previewID); err != nil {
			log.Printf("Failed to destroy preview for PR #%d: %v", event.Number, err)
		}
	}
}

func extractService(branch string) string {
	if branch == "" {
		return "default"
	}
	return branch
}

func verifySignature(payload []byte, signature, secret string) bool {
	if signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expected))
}
