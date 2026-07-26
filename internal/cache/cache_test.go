package cache

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"deploymate/internal/model"
)

// mockCache implements the Cache interface using in-memory maps.
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

func (m *mockCache) Close() error {
	return nil
}

func TestCache_Get(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		dest    interface{}
		setup   func(c *mockCache)
		wantErr bool
	}{
		{
			name: "hit returns previously set value",
			key:  "key-hit",
			dest: new(model.DeploymentSpec),
			setup: func(c *mockCache) {
				spec := model.DeploymentSpec{
					ID:          "dep-1",
					OrgID:       "org-1",
					ProjectID:   "proj-1",
					Environment: "production",
					Service:     "api",
					Image:       "gcr.io/proj/api:v1",
					Replicas:    3,
					Resources: model.ResourceSpec{
						CPU:    "500m",
						Memory: "256Mi",
					},
					EnvVars: map[string]string{"LOG_LEVEL": "info"},
				}
				data, _ := json.Marshal(spec)
				c.store["key-hit"] = string(data)
			},
			wantErr: false,
		},
		{
			name: "miss returns error",
			key:  "key-miss",
			dest: new(model.DeploymentSpec),
			setup: func(c *mockCache) {
				// store remains empty
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockCache()
			tt.setup(mock)

			err := mock.Get(context.Background(), tt.key, tt.dest)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				spec := tt.dest.(*model.DeploymentSpec)
				if spec.ID != "dep-1" {
					t.Errorf("Get() got ID = %q, want %q", spec.ID, "dep-1")
				}
			}
		})
	}
}

func TestCache_Set(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   interface{}
		wantErr bool
	}{
		{
			name: "set stores DeploymentSpec",
			key:  "set-spec",
			value: &model.DeploymentSpec{
				ID:    "dep-2",
				OrgID: "org-2",
			},
			wantErr: false,
		},
		{
			name: "set stores plain string",
			key:  "set-string",
			value: "hello",
			wantErr: false,
		},
		{
			name: "set stores map",
			key:  "set-map",
			value: map[string]string{"a": "1", "b": "2"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockCache()

			err := mock.Set(context.Background(), tt.key, tt.value, 5*time.Second)
			if (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if _, ok := mock.store[tt.key]; !ok {
					t.Errorf("Set() key %q not found in store", tt.key)
				}
			}
		})
	}
}

func TestCache_Get_Set_RoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		signal *model.DeploymentSpec
	}{
		{
			name: "round-trip with full spec",
			key:  "rt-full",
			signal: &model.DeploymentSpec{
				ID:          "dep-rt",
				OrgID:       "org-rt",
				ProjectID:   "proj-rt",
				Environment: "staging",
				Service:     "worker",
				Image:       "gcr.io/proj/worker:v2",
				Replicas:    5,
				Resources: model.ResourceSpec{
					CPU:    "1000m",
					Memory: "512Mi",
				},
				EnvVars:   map[string]string{"ENV": "staging", "DEBUG": "true"},
				PolicyRef: "policy-1",
				TargetRef: "target-1",
			},
		},
		{
			name: "round-trip with minimal spec",
			key:  "rt-min",
			signal: &model.DeploymentSpec{
				ID:    "dep-min",
				OrgID: "org-min",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockCache()
			ctx := context.Background()

			if err := mock.Set(ctx, tt.key, tt.signal, 5*time.Second); err != nil {
				t.Fatalf("Set() unexpected error: %v", err)
			}

			got := new(model.DeploymentSpec)
			if err := mock.Get(ctx, tt.key, got); err != nil {
				t.Fatalf("Get() unexpected error: %v", err)
			}

			if got.ID != tt.signal.ID {
				t.Errorf("ID = %q, want %q", got.ID, tt.signal.ID)
			}
			if got.OrgID != tt.signal.OrgID {
				t.Errorf("OrgID = %q, want %q", got.OrgID, tt.signal.OrgID)
			}
			if got.Replicas != tt.signal.Replicas {
				t.Errorf("Replicas = %d, want %d", got.Replicas, tt.signal.Replicas)
			}
			if got.EnvVars["ENV"] != tt.signal.EnvVars["ENV"] {
				t.Errorf("EnvVars[ENV] = %q, want %q", got.EnvVars["ENV"], tt.signal.EnvVars["ENV"])
			}
		})
	}
}

func TestCache_Del(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		setup    func(c *mockCache)
		wantErr  bool
		wantMiss bool
	}{
		{
			name: "delete existing key",
			key:  "del-exist",
			setup: func(c *mockCache) {
				c.store["del-exist"] = `"data"`
			},
			wantErr:  false,
			wantMiss: true,
		},
		{
			name: "delete non-existing key",
			key:  "del-missing",
			setup: func(c *mockCache) {
				// empty
			},
			wantErr:  false,
			wantMiss: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockCache()
			tt.setup(mock)

			err := mock.Del(context.Background(), tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Del() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantMiss {
				if _, ok := mock.store[tt.key]; ok {
					t.Errorf("Del() key %q still present after deletion", tt.key)
				}
			}
		})
	}
}

func TestCache_Del_Get_Miss_RoundTrip(t *testing.T) {
	mock := newMockCache()
	ctx := context.Background()

	spec := &model.DeploymentSpec{ID: "dep-del", OrgID: "org-del"}
	_ = mock.Set(ctx, "del-rt-key", spec, 5*time.Second)

	// Confirm it exists
	var before model.DeploymentSpec
	if err := mock.Get(ctx, "del-rt-key", &before); err != nil {
		t.Fatalf("Get() before Del failed: %v", err)
	}
	if before.ID != "dep-del" {
		t.Fatalf("Get() before Del got ID = %q, want %q", before.ID, "dep-del")
	}

	// Delete
	if err := mock.Del(ctx, "del-rt-key"); err != nil {
		t.Fatalf("Del() error: %v", err)
	}

	// Confirm miss
	var after model.DeploymentSpec
	if err := mock.Get(ctx, "del-rt-key", &after); err == nil {
		t.Errorf("Get() after Del succeeded, want error")
	}
}

func TestGetDesiredState_CacheHit(t *testing.T) {
	mock := newMockCache()
	ctx := context.Background()
	orgID := "org-1"
	deploymentID := "dep-1"

	// Pre-populate cache
	spec := &model.DeploymentSpec{
		ID:    deploymentID,
		OrgID: orgID,
		Image: "gcr.io/proj/api:v1",
	}
	key := "deploymate:desired:" + orgID + ":" + deploymentID
	data, _ := json.Marshal(spec)
	mock.store[key] = string(data)

	got, hit := getFromCache(mock, ctx, orgID, deploymentID)
	if !hit {
		t.Fatal("getFromCache() expected hit, got miss")
	}
	if got.ID != deploymentID {
		t.Errorf("getFromCache() ID = %q, want %q", got.ID, deploymentID)
	}
}

func TestGetDesiredState_CacheMiss(t *testing.T) {
	mock := newMockCache()
	ctx := context.Background()

	got, hit := getFromCache(mock, ctx, "org-1", "dep-miss")
	if hit {
		t.Error("getFromCache() expected miss, got hit")
	}
	if got != nil {
		t.Errorf("getFromCache() returned non-nil spec on miss: %+v", got)
	}
}

func TestSetDesiredState(t *testing.T) {
	mock := newMockCache()
	ctx := context.Background()
	orgID := "org-1"
	deploymentID := "dep-1"

	spec := &model.DeploymentSpec{
		ID:          deploymentID,
		OrgID:       orgID,
		Environment: "production",
		Replicas:    3,
	}

	if err := setCache(mock, ctx, orgID, deploymentID, spec); err != nil {
		t.Fatalf("setCache() error: %v", err)
	}

	key := "deploymate:desired:" + orgID + ":" + deploymentID
	if _, ok := mock.store[key]; !ok {
		t.Errorf("setCache() key %q not found in store", key)
	}
}

func TestInvalidateDesiredState(t *testing.T) {
	mock := newMockCache()
	ctx := context.Background()
	orgID := "org-1"
	deploymentID := "dep-1"

	key := "deploymate:desired:" + orgID + ":" + deploymentID
	mock.store[key] = `"data"`

	if err := mock.Del(ctx, key); err != nil {
		t.Fatalf("Del() error: %v", err)
	}

	if _, ok := mock.store[key]; ok {
		t.Errorf("Del() key %q still present after invalidation", key)
	}
}

// Helper functions that mirror the handler's cache access patterns.
func getFromCache(c Cache, ctx context.Context, orgID, deploymentID string) (*model.DeploymentSpec, bool) {
	key := "deploymate:desired:" + orgID + ":" + deploymentID
	var spec model.DeploymentSpec
	if err := c.Get(ctx, key, &spec); err != nil {
		return nil, false
	}
	return &spec, true
}

func setCache(c Cache, ctx context.Context, orgID, deploymentID string, spec *model.DeploymentSpec) error {
	key := "deploymate:desired:" + orgID + ":" + deploymentID
	return c.Set(ctx, key, spec, 5*time.Second)
}
