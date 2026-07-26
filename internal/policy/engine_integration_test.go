package policy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEngine_LoadBundle_FromHTTPServer(t *testing.T) {
	bundle := []map[string]interface{}{
		{
			"name": "cpu-limit",
			"rules": []map[string]interface{}{
				{
					"condition": map[string]interface{}{"cpu": "4"},
					"action":    "deny",
					"message":   "CPU limit too high",
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bundle)
	}))
	defer srv.Close()

	engine := NewEngine(5 * time.Minute)
	if err := engine.LoadBundle(context.Background(), srv.URL); err != nil {
		t.Fatalf("LoadBundle() error = %v", err)
	}

	if !engine.IsLoaded() {
		t.Fatal("Expected bundle to be loaded")
	}

	if engine.BundleHash() == "" {
		t.Fatal("Expected non-empty bundle hash")
	}
}

func TestEngine_Evaluate_ComplexPolicy(t *testing.T) {
	bundle := []map[string]interface{}{
		{
			"name": "multi-rule",
			"rules": []map[string]interface{}{
				{
					"condition": map[string]interface{}{"environment": "production", "has_approval": "false"},
					"action":    "deny",
					"message":   "Production requires approval",
				},
				{
					"condition": map[string]interface{}{"cpu": "8"},
					"action":    "deny",
					"message":   "CPU too high",
				},
				{
					"condition": map[string]interface{}{"memory": "32Gi"},
					"action":    "deny",
					"message":   "Memory too high",
				},
			},
		},
	}

	engine := NewEngine(5 * time.Minute)
	engine.mu.Lock()
	data, _ := json.Marshal(bundle)
	engine.bundle = data
	engine.loaded = true
	engine.loadedAt = time.Now()
	engine.mu.Unlock()

	tests := []struct {
		name     string
		input    map[string]interface{}
		allowed  bool
		violations int
	}{
		{
			name:     "production without approval",
			input:    map[string]interface{}{"environment": "production", "has_approval": "false"},
			allowed:  false,
			violations: 1,
		},
		{
			name:     "production with approval",
			input:    map[string]interface{}{"environment": "production", "has_approval": "true"},
			allowed:  true,
			violations: 0,
		},
		{
			name:     "staging without approval",
			input:    map[string]interface{}{"environment": "staging", "has_approval": "false"},
			allowed:  true,
			violations: 0,
		},
		{
			name:     "high CPU",
			input:    map[string]interface{}{"cpu": "8"},
			allowed:  false,
			violations: 1,
		},
		{
			name:     "high memory",
			input:    map[string]interface{}{"memory": "32Gi"},
			allowed:  false,
			violations: 1,
		},
		{
			name:     "normal resources",
			input:    map[string]interface{}{"cpu": "2", "memory": "4Gi"},
			allowed:  true,
			violations: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Evaluate(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if result.Allowed != tt.allowed {
				t.Errorf("Allowed = %v, want %v", result.Allowed, tt.allowed)
			}
			if len(result.Violations) != tt.violations {
				t.Errorf("Violations = %d, want %d", len(result.Violations), tt.violations)
			}
		})
	}
}

func TestEngine_BundleExpiry(t *testing.T) {
	engine := NewEngine(1 * time.Millisecond)
	engine.mu.Lock()
	engine.bundle = []byte(`[{"name":"test","rules":[]}]`)
	engine.loaded = true
	engine.loadedAt = time.Now()
	engine.mu.Unlock()

	time.Sleep(5 * time.Millisecond)

	_, err := engine.Evaluate(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Fatal("Expected error for expired bundle")
	}
}
