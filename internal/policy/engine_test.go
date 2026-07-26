package policy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEvaluate_Allowed(t *testing.T) {
	bundle := `[{"name":"cpu-limit","rules":[{"condition":{"cpu":"1"},"action":"allow","message":"CPU within limit"}]}]`
	engine := setupEngine(t, bundle)

	input := map[string]interface{}{"cpu": "1", "memory": "512Mi"}
	result, err := engine.Evaluate(context.Background(), input)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !result.Allowed {
		t.Error("Expected allowed=true")
	}
}

func TestEvaluate_Denied(t *testing.T) {
	bundle := `[{"name":"prod-guard","rules":[{"condition":{"environment":"production","needs_approval":"false"},"action":"deny","message":"Production deploys require approval"}]}]`
	engine := setupEngine(t, bundle)

	input := map[string]interface{}{"environment": "production", "needs_approval": "false"}
	result, err := engine.Evaluate(context.Background(), input)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if result.Allowed {
		t.Error("Expected allowed=false")
	}
	if len(result.Violations) != 1 {
		t.Errorf("Expected 1 violation, got %d", len(result.Violations))
	}
}

func TestEvaluate_NoBundle(t *testing.T) {
	engine := NewEngine(5 * time.Minute)
	_, err := engine.Evaluate(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Fatal("Expected error for no bundle loaded")
	}
}

func TestEvaluate_ExpiredBundle(t *testing.T) {
	bundle := `[{"name":"test","rules":[]}]`
	engine := setupEngine(t, bundle)
	engine.mu.Lock()
	engine.loadedAt = time.Now().Add(-10 * time.Minute)
	engine.mu.Unlock()

	_, err := engine.Evaluate(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Fatal("Expected error for expired bundle")
	}
}

func TestLoadBundle_FromFile(t *testing.T) {
	bundle := `[{"name":"file-test","rules":[{"condition":{"source":"file"},"action":"deny","message":"loaded from file"}]}]`
	path := filepath.Join(t.TempDir(), "bundle.json")
	os.WriteFile(path, []byte(bundle), 0644)

	engine := NewEngine(5 * time.Minute)
	if err := engine.LoadBundle(context.Background(), path); err != nil {
		t.Fatalf("LoadBundle() error = %v", err)
	}
	if !engine.IsLoaded() {
		t.Error("Expected bundle to be loaded")
	}
}

func TestLoadBundle_FromURL(t *testing.T) {
	bundle := `[{"name":"url-test","rules":[]}]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(json.RawMessage(bundle))
	}))
	defer srv.Close()

	engine := NewEngine(5 * time.Minute)
	if err := engine.LoadBundle(context.Background(), srv.URL); err != nil {
		t.Fatalf("LoadBundle() error = %v", err)
	}
	if !engine.IsLoaded() {
		t.Error("Expected bundle to be loaded")
	}
}

func TestBundleHash(t *testing.T) {
	bundle := `[{"name":"hash-test","rules":[]}]`
	engine := setupEngine(t, bundle)
	hash := engine.BundleHash()
	if hash == "" {
		t.Error("Expected non-empty bundle hash")
	}
}

func setupEngine(t *testing.T, bundleJSON string) *Engine {
	t.Helper()
	engine := NewEngine(5 * time.Minute)
	hash := sha256.Sum256([]byte(bundleJSON))
	engine.mu.Lock()
	engine.bundle = []byte(bundleJSON)
	engine.bundleHash = hex.EncodeToString(hash[:])
	engine.loaded = true
	engine.loadedAt = time.Now()
	engine.mu.Unlock()
	return engine
}
