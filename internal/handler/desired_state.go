package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"deploymate/internal/cache"
	"deploymate/internal/model"
	"deploymate/internal/store"
)

type DesiredStateHandler struct {
	cache cache.Cache
	store *store.Store
}

func NewDesiredStateHandler(cache cache.Cache, store *store.Store) *DesiredStateHandler {
	return &DesiredStateHandler{
		cache: cache,
		store: store,
	}
}

func (h *DesiredStateHandler) GetDesiredState(w http.ResponseWriter, r *http.Request) {
	deploymentID := chi.URLParam(r, "id")
	if deploymentID == "" {
		http.Error(w, "deployment ID required", http.StatusBadRequest)
		return
	}

	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		http.Error(w, "organization ID required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	if spec, hit := h.getFromCache(ctx, orgID, deploymentID); hit {
		w.Header().Set("X-Cache", "HIT")
		writeJSON(w, spec)
		return
	}

	spec, err := h.store.GetDesiredState(ctx, orgID, deploymentID)
	if err != nil {
		http.Error(w, "deployment not found", http.StatusNotFound)
		return
	}

	h.setCache(ctx, orgID, deploymentID, spec)

	w.Header().Set("X-Cache", "MISS")
	writeJSON(w, spec)
}

func (h *DesiredStateHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	deploymentID := chi.URLParam(r, "id")
	if deploymentID == "" {
		http.Error(w, "deployment ID required", http.StatusBadRequest)
		return
	}

	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		http.Error(w, "organization ID required", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"deployment_id": deploymentID,
		"status":        "rollback_initiated",
		"message":       "Rollback triggered, agent will reconcile to previous image",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *DesiredStateHandler) Events(w http.ResponseWriter, r *http.Request) {
	deploymentID := r.URL.Query().Get("deployment_id")
	if deploymentID == "" {
		http.Error(w, "deployment_id query parameter required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()

	sendEvent(w, "connected", map[string]interface{}{
		"deployment_id": deploymentID,
		"timestamp":     time.Now().Unix(),
	})
	flusher.Flush()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sendEvent(w, "heartbeat", map[string]interface{}{
				"deployment_id": deploymentID,
				"timestamp":     time.Now().Unix(),
			})
			flusher.Flush()
		}
	}
}

func (h *DesiredStateHandler) getFromCache(ctx context.Context, orgID, deploymentID string) (*model.DeploymentSpec, bool) {
	key := "deploymate:desired:" + orgID + ":" + deploymentID
	var spec model.DeploymentSpec
	if err := h.cache.Get(ctx, key, &spec); err != nil {
		return nil, false
	}
	return &spec, true
}

func (h *DesiredStateHandler) setCache(ctx context.Context, orgID, deploymentID string, spec *model.DeploymentSpec) {
	key := "deploymate:desired:" + orgID + ":" + deploymentID
	h.cache.Set(ctx, key, spec, 5*time.Second)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func sendEvent(w http.ResponseWriter, eventType string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	w.Write([]byte("event: " + eventType + "\n"))
	w.Write([]byte("data: " + string(jsonData) + "\n\n"))
}

func RegisterRoutes(r chi.Router, handler *DesiredStateHandler) {
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.RequestID)
		r.Use(middleware.RealIP)
		r.Use(middleware.Logger)
		r.Use(middleware.Recoverer)

		r.Get("/deployments/{id}/desired-state", handler.GetDesiredState)
	})
}
