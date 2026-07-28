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

func (h *DesiredStateHandler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		http.Error(w, "organization ID required", http.StatusBadRequest)
		return
	}

	envFilter := r.URL.Query().Get("env")

	specs, err := h.store.ListDeployments(r.Context(), orgID, envFilter)
	if err != nil {
		http.Error(w, "failed to list deployments", http.StatusInternalServerError)
		return
	}

	writeJSON(w, specs)
}

func (h *DesiredStateHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		http.Error(w, "organization ID required", http.StatusBadRequest)
		return
	}

	agents, err := h.store.ListAgents(r.Context(), orgID)
	if err != nil {
		http.Error(w, "failed to list agents", http.StatusInternalServerError)
		return
	}

	writeJSON(w, agents)
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

	ctx := r.Context()

	// Invalidate cache to force agent to fetch new desired state
	h.invalidateCache(ctx, orgID, deploymentID)

	response := map[string]interface{}{
		"deployment_id": deploymentID,
		"status":        "rollback_initiated",
		"message":       "Rollback triggered, agent will reconcile to previous image",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *DesiredStateHandler) UpdateDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	deploymentID := chi.URLParam(r, "id")
	if deploymentID == "" {
		http.Error(w, "deployment ID required", http.StatusBadRequest)
		return
	}

	agentID := r.Header.Get("X-Agent-ID")
	if agentID == "" {
		http.Error(w, "agent ID required", http.StatusBadRequest)
		return
	}

	var status model.DeploymentStatus
	if err := json.NewDecoder(r.Body).Decode(&status); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	status.ID = deploymentID

	// Store status update
	if err := h.store.UpdateDeploymentStatus(r.Context(), deploymentID, &status); err != nil {
		http.Error(w, "failed to update status", http.StatusInternalServerError)
		return
	}

	// Broadcast status update via SSE
	h.broadcastEvent(deploymentID, "deployment.status", map[string]interface{}{
		"deployment_id": deploymentID,
		"agent_id":      agentID,
		"phase":         status.Phase,
		"message":       status.Message,
		"progress":      status.ProgressPct,
		"timestamp":     time.Now().Unix(),
	})

	writeJSON(w, map[string]string{"status": "updated"})
}

func (h *DesiredStateHandler) AgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "id")
	if agentID == "" {
		http.Error(w, "agent ID required", http.StatusBadRequest)
		return
	}

	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		http.Error(w, "organization ID required", http.StatusBadRequest)
		return
	}

	// Update agent heartbeat
	if err := h.store.UpdateAgentHeartbeat(r.Context(), agentID, orgID); err != nil {
		http.Error(w, "failed to update heartbeat", http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

func (h *DesiredStateHandler) GetDeploymentEvents(w http.ResponseWriter, r *http.Request) {
	deploymentID := chi.URLParam(r, "id")
	if deploymentID == "" {
		http.Error(w, "deployment ID required", http.StatusBadRequest)
		return
	}

	events, err := h.store.GetDeploymentEvents(r.Context(), deploymentID, 50)
	if err != nil {
		http.Error(w, "failed to get events", http.StatusInternalServerError)
		return
	}

	writeJSON(w, events)
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

func (h *DesiredStateHandler) invalidateCache(ctx context.Context, orgID, deploymentID string) {
	key := "deploymate:desired:" + orgID + ":" + deploymentID
	h.cache.Del(ctx, key)
}

func (h *DesiredStateHandler) broadcastEvent(deploymentID, eventType string, data interface{}) {
	// TODO: Implement SSE broadcast to connected clients
	// For now, just log the event
	_ = deploymentID
	_ = eventType
	_ = data
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

		r.Get("/deployments", handler.ListDeployments)
		r.Get("/deployments/{id}/desired-state", handler.GetDesiredState)
		r.Post("/deployments/{id}/rollback", handler.Rollback)
		r.Get("/agents", handler.ListAgents)
		r.Get("/events", handler.Events)
	})
}
