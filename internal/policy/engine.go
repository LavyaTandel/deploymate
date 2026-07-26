package policy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// PolicyResult is the outcome of a policy evaluation
type PolicyResult struct {
	Allowed    bool     `json:"allowed"`
	Violations []string `json:"violations,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

// Engine evaluates deployment policies against input
type Engine struct {
	mu         sync.RWMutex
	bundle     []byte
	bundleHash string
	loaded     bool
	loadedAt   time.Time
	ttl        time.Duration
}

// NewEngine creates a policy engine with bundle TTL
func NewEngine(ttl time.Duration) *Engine {
	if ttl == 0 {
		ttl = 5 * time.Minute
	}
	return &Engine{ttl: ttl}
}

// LoadBundle loads a policy bundle from file path or URL
func (e *Engine) LoadBundle(ctx context.Context, source string) error {
	var data []byte
	var err error

	if _, statErr := os.Stat(source); statErr == nil {
		data, err = os.ReadFile(source)
	} else {
		data, err = fetchBundle(ctx, source)
	}
	if err != nil {
		return fmt.Errorf("load bundle: %w", err)
	}

	hash := sha256.Sum256(data)
	e.mu.Lock()
	e.bundle = data
	e.bundleHash = hex.EncodeToString(hash[:])
	e.loaded = true
	e.loadedAt = time.Now()
	e.mu.Unlock()
	return nil
}

// Evaluate checks input against loaded policy bundle
func (e *Engine) Evaluate(ctx context.Context, input map[string]interface{}) (*PolicyResult, error) {
	e.mu.RLock()
	if !e.loaded {
		e.mu.RUnlock()
		return nil, fmt.Errorf("no policy bundle loaded")
	}
	if time.Since(e.loadedAt) > e.ttl {
		e.mu.RUnlock()
		return nil, fmt.Errorf("policy bundle expired")
	}
	bundle := e.bundle
	e.mu.RUnlock()

	return evaluateBundle(bundle, input)
}

// IsLoaded returns whether a bundle is loaded and fresh
func (e *Engine) IsLoaded() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.loaded
}

// BundleHash returns SHA256 of loaded bundle
func (e *Engine) BundleHash() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.bundleHash
}

func fetchBundle(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d fetching bundle", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func evaluateBundle(bundle []byte, input map[string]interface{}) (*PolicyResult, error) {
	var policies []struct {
		Name  string `json:"name"`
		Rules []struct {
			Condition map[string]interface{} `json:"condition"`
			Action    string                 `json:"action"`
			Message   string                 `json:"message"`
		} `json:"rules"`
	}

	if err := json.Unmarshal(bundle, &policies); err != nil {
		return nil, fmt.Errorf("parse bundle: %w", err)
	}

	result := &PolicyResult{
		Allowed: true,
		Details: make(map[string]interface{}),
	}

	for _, policy := range policies {
		for _, rule := range policy.Rules {
			if matchCondition(rule.Condition, input) {
				if rule.Action == "deny" {
					result.Allowed = false
					result.Violations = append(result.Violations, rule.Message)
				}
			}
		}
	}

	return result, nil
}

func matchCondition(condition, input map[string]interface{}) bool {
	for key, expected := range condition {
		actual, ok := input[key]
		if !ok {
			return false
		}
		if fmt.Sprintf("%v", expected) != fmt.Sprintf("%v", actual) {
			return false
		}
	}
	return true
}
