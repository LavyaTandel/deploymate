package preview

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Manager handles preview environment lifecycle
type Manager struct {
	mu        sync.RWMutex
	previewEnv map[string]*PreviewEnv
}

// NewManager creates a new preview manager
func NewManager() *Manager {
	return &Manager{
		previewEnv: make(map[string]*PreviewEnv),
	}
}

// CreatePreview creates a new preview environment from a PR request
func (m *Manager) CreatePreview(ctx context.Context, req PreviewRequest) (*PreviewEnv, error) {
	previewID := fmt.Sprintf("%s-pr-%d", req.Service, req.PRNumber)

	m.mu.Lock()
	if _, exists := m.previewEnv[previewID]; exists {
		m.mu.Unlock()
		return nil, fmt.Errorf("preview %s already exists", previewID)
	}

	env := &PreviewEnv{
		ID:        previewID,
		PRNumber:  req.PRNumber,
		URL:       fmt.Sprintf("https://%s.preview.deploymate.io", previewID),
		Status:    StatusCreating,
		CreatedAt: time.Now(),
	}
	m.previewEnv[previewID] = env
	m.mu.Unlock()

	// Simulate async creation
	go func() {
		time.Sleep(2 * time.Second)
		m.mu.Lock()
		env.Status = StatusRunning
		m.mu.Unlock()
		log.Printf("Preview %s created at %s", previewID, env.URL)
	}()

	return env, nil
}

// DestroyPreview destroys a preview environment
func (m *Manager) DestroyPreview(ctx context.Context, previewID string) error {
	m.mu.Lock()
	env, exists := m.previewEnv[previewID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("preview %s not found", previewID)
	}
	env.Status = StatusDestroying
	m.mu.Unlock()

	// Simulate async destruction
	go func() {
		time.Sleep(1 * time.Second)
		m.mu.Lock()
		env.Status = StatusDestroyed
		m.mu.Unlock()
		log.Printf("Preview %s destroyed", previewID)
	}()

	return nil
}

// ListPreviews returns all preview environments for a service
func (m *Manager) ListPreviews(ctx context.Context, service string) ([]*PreviewEnv, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*PreviewEnv
	for _, env := range m.previewEnv {
		if service == "" || env.ID[:len(env.ID)-len(fmt.Sprintf("-pr-%d", env.PRNumber))] == service {
			results = append(results, env)
		}
	}
	return results, nil
}
