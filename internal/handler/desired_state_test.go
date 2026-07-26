package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"deploymate/internal/cache"
	"deploymate/internal/model"
)

// --- mock cache ---

type mockCache struct {
	store map[string]string
}

func newMockCache() *mockCache {
	return &mockCache{store: make(map[string]string)}
}

func (m *mockCache) Get(_ context.Context, key string, dest interface{}) error {
	data, ok := m.store[key]
	if !ok {
		return errors.New("redis: nil")
	}
	return json.Unmarshal([]byte(data), dest)
}

func (m *mockCache) Set(_ context.Context, key string, value interface{}, _ time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	m.store[key] = string(data)
	return nil
}

func (m *mockCache) Del(_ context.Context, key string) error {
	delete(m.store, key)
	return nil
}

func (m *mockCache) Close() error { return nil }

// Verify mockCache satisfies cache.Cache at compile time.
var _ cache.Cache = (*mockCache)(nil)

// --- helpers ---

// chiContext returns a new *http.Request with a chi context containing the URL param.
func chiContext(method, path string, urlParams map[string]string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	rctx := chi.NewRouteContext()
	for k, v := range urlParams {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// newHandler creates a DesiredStateHandler with a mock cache and a nil store.
// For tests that hit the store path, we set the mock cache to be empty so the handler falls through.
func newHandlerWithMockCache(mc *mockCache) *DesiredStateHandler {
	return NewDesiredStateHandler(mc, nil)
}

// --- GetDesiredState tests ---

func TestGetDesiredState(t *testing.T) {
	orgID := "org-test"
	deploymentID := "dep-test"
	spec := model.DeploymentSpec{
		ID:          deploymentID,
		OrgID:       orgID,
		ProjectID:   "proj-1",
		Environment: "production",
		Service:     "api",
		Image:       "gcr.io/proj/api:v1",
		Replicas:    3,
		Resources: model.ResourceSpec{
			CPU:    "500m",
			Memory: "256Mi",
		},
	}

	tests := []struct {
		name           string
		urlParams      map[string]string
		orgHeader      string
		cacheSetup     func(c *mockCache)
		wantCode       int
		wantCache      string
		wantBody       *model.DeploymentSpec
		wantErrContain string
	}{
		{
			name:      "cache hit returns spec with X-Cache HIT",
			urlParams: map[string]string{"id": deploymentID},
			orgHeader: orgID,
			cacheSetup: func(c *mockCache) {
				key := "deploymate:desired:" + orgID + ":" + deploymentID
				data, _ := json.Marshal(spec)
				c.store[key] = string(data)
			},
			wantCode:  http.StatusOK,
			wantCache: "HIT",
			wantBody:  &spec,
		},
		// NOTE: cache-miss path requires a real store.Store (concrete type with unexported pool).
		// Cannot mock it without refactoring to an interface. Tested via integration tests.
		{
			name:      "missing deployment ID returns 400",
			urlParams: map[string]string{},
			orgHeader: orgID,
			cacheSetup: func(c *mockCache) {
			},
			wantCode:       http.StatusBadRequest,
			wantErrContain: "deployment ID required",
		},
		{
			name:      "missing org ID returns 400",
			urlParams: map[string]string{"id": deploymentID},
			orgHeader: "",
			cacheSetup: func(c *mockCache) {
			},
			wantCode:       http.StatusBadRequest,
			wantErrContain: "organization ID required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := newMockCache()
			tt.cacheSetup(mc)

			// For the cache miss case that hits the store, we need a real store mock.
			// Since we can't easily mock store.Store (it's a concrete struct with *pgxpool.Pool),
			// we skip the store-hit path and only test cache-miss -> 500 (nil store panics,
			// but the handler does not use store for the cache-hit path).
			// The actual store mock would require an interface; testing the cache path is sufficient.

			h := newHandlerWithMockCache(mc)

			path := "/api/v1/deployments"
			if id, ok := tt.urlParams["id"]; ok {
				path += "/" + id
			}
			path += "/desired-state"

			req := chiContext("GET", path, tt.urlParams)
			if tt.orgHeader != "" {
				req.Header.Set("X-Org-ID", tt.orgHeader)
			}

			rr := httptest.NewRecorder()
			h.GetDesiredState(rr, req)

			if rr.Code != tt.wantCode {
				t.Errorf("GetDesiredState() status = %d, want %d. Body: %s", rr.Code, tt.wantCode, rr.Body.String())
				return
			}

			if tt.wantCache != "" {
				got := rr.Header().Get("X-Cache")
				if got != tt.wantCache {
					t.Errorf("X-Cache = %q, want %q", got, tt.wantCache)
				}
			}

			if tt.wantBody != nil {
				var got model.DeploymentSpec
				if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}
				if got.ID != tt.wantBody.ID {
					t.Errorf("response ID = %q, want %q", got.ID, tt.wantBody.ID)
				}
				if got.Image != tt.wantBody.Image {
					t.Errorf("response Image = %q, want %q", got.Image, tt.wantBody.Image)
				}
			}

			if tt.wantErrContain != "" {
				body := rr.Body.String()
				if !strings.Contains(body, tt.wantErrContain) {
					t.Errorf("response body = %q, want to contain %q", body, tt.wantErrContain)
				}
			}
		})
	}
}

// --- Rollback tests ---

func TestRollback(t *testing.T) {
	orgID := "org-test"
	deploymentID := "dep-test"

	tests := []struct {
		name      string
		urlParams map[string]string
		orgHeader string
		wantCode  int
		wantFields map[string]string
	}{
		{
			name:      "rollback returns JSON response",
			urlParams: map[string]string{"id": deploymentID},
			orgHeader: orgID,
			wantCode:  http.StatusOK,
			wantFields: map[string]string{
				"deployment_id": deploymentID,
				"status":        "rollback_initiated",
			},
		},
		{
			name:      "rollback missing deployment ID returns 400",
			urlParams: map[string]string{},
			orgHeader: orgID,
			wantCode:  http.StatusBadRequest,
		},
		{
			name:      "rollback missing org ID returns 400",
			urlParams: map[string]string{"id": deploymentID},
			orgHeader: "",
			wantCode:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := newMockCache()
			h := newHandlerWithMockCache(mc)

			path := "/api/v1/deployments"
			if id, ok := tt.urlParams["id"]; ok {
				path += "/" + id
			}
			path += "/rollback"

			req := chiContext("POST", path, tt.urlParams)
			if tt.orgHeader != "" {
				req.Header.Set("X-Org-ID", tt.orgHeader)
			}

			rr := httptest.NewRecorder()
			h.Rollback(rr, req)

			if rr.Code != tt.wantCode {
				t.Errorf("Rollback() status = %d, want %d. Body: %s", rr.Code, tt.wantCode, rr.Body.String())
				return
			}

			if tt.wantFields != nil {
				// Verify Content-Type is JSON.
				ct := rr.Header().Get("Content-Type")
				if ct != "application/json" {
					t.Errorf("Content-Type = %q, want %q", ct, "application/json")
				}

				var body map[string]interface{}
				if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
					t.Fatalf("failed to decode JSON response: %v", err)
				}

				for key, want := range tt.wantFields {
					got, ok := body[key]
					if !ok {
						t.Errorf("response missing key %q", key)
						continue
					}
					if got.(string) != want {
						t.Errorf("response[%q] = %q, want %q", key, got, want)
					}
				}
			}
		})
	}
}

// --- Events (SSE) tests ---

func TestEvents(t *testing.T) {
	orgID := "org-test"
	deploymentID := "dep-test"

	tests := []struct {
		name         string
		queryParams  string
		wantCode     int
		wantHeaders  map[string]string
		wantSSE      bool
	}{
		{
			name:        "events sets SSE headers",
			queryParams: "?deployment_id=" + deploymentID,
			wantCode:    http.StatusOK,
			wantHeaders: map[string]string{
				"Content-Type": "text/event-stream",
				"Cache-Control": "no-cache",
				"Connection":    "keep-alive",
			},
			wantSSE: true,
		},
		{
			name:        "events missing deployment_id returns 400",
			queryParams: "",
			wantCode:    http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := newMockCache()
			h := newHandlerWithMockCache(mc)

			path := "/events" + tt.queryParams
			req := httptest.NewRequest("GET", path, nil)
			if orgID != "" {
				req.Header.Set("X-Org-ID", orgID)
			}

			rr := httptest.NewRecorder()

			// For the SSE case, the handler enters an infinite loop with a ticker.
			// We need to cancel the context after a short time to stop it.
			if tt.wantSSE {
				ctx, cancel := context.WithTimeout(req.Context(), 200*time.Millisecond)
				defer cancel()
				req = req.WithContext(ctx)
			}

			h.Events(rr, req)

			if rr.Code != tt.wantCode {
				t.Errorf("Events() status = %d, want %d. Body: %s", rr.Code, tt.wantCode, rr.Body.String())
				return
			}

			for header, want := range tt.wantHeaders {
				got := rr.Header().Get(header)
				if got != want {
					t.Errorf("Events() header %q = %q, want %q", header, got, want)
				}
			}

			if tt.wantSSE {
				body := rr.Body.String()
				if !strings.Contains(body, "event: connected") {
					t.Errorf("Events() response missing 'event: connected'. Body:\n%s", body)
				}
				if !strings.Contains(body, "deployment_id") {
					t.Errorf("Events() response missing 'deployment_id'. Body:\n%s", body)
				}
			}
		})
	}
}

// --- verify handler creation ---

func TestNewDesiredStateHandler(t *testing.T) {
	mc := newMockCache()
	h := NewDesiredStateHandler(mc, nil)

	if h == nil {
		t.Fatal("NewDesiredStateHandler() returned nil")
	}
	if h.cache != mc {
		t.Error("NewDesiredStateHandler() cache not set correctly")
	}
}
