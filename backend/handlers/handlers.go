package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"fact-checker/models"
	"fact-checker/services"

	"github.com/go-chi/chi/v5"
)

// Handler contains all HTTP handlers
type Handler struct {
	factChecker *services.FactCheckService
}

// NewHandler creates a new handler instance
func NewHandler(fc *services.FactCheckService) *Handler {
	return &Handler{
		factChecker: fc,
	}
}

// HealthCheck returns service health status
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "factchecker-api",
	})
}

// CheckFacts initiates a new fact-check
func (h *Handler) CheckFacts(w http.ResponseWriter, r *http.Request) {
	var req models.CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	check, err := h.factChecker.StartCheck(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(check)
}

// GetCheckStatus returns the current status of a check
func (h *Handler) GetCheckStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	check, ok := h.factChecker.GetCheck(id)
	if !ok {
		http.Error(w, "Check not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(check)
}

// StreamCheckProgress streams progress updates via SSE
func (h *Handler) StreamCheckProgress(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Check if the check exists
	_, ok := h.factChecker.GetCheck(id)
	if !ok {
		http.Error(w, "Check not found", http.StatusNotFound)
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Subscribe to updates
	ch := h.factChecker.Subscribe(id)
	defer h.factChecker.Unsubscribe(id, ch)

	// Send initial status
	check, _ := h.factChecker.GetCheck(id)
	data, _ := json.Marshal(check)
	fmt.Fprintf(w, "event: status\ndata: %s\n\n", data)
	flusher.Flush()

	// Stream updates
	for {
		select {
		case update, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(update)
			fmt.Fprintf(w, "event: progress\ndata: %s\n\n", data)
			flusher.Flush()

			// Check if complete
			if update.Step == "complete" || update.Step == "error" {
				// Send final status
				check, _ := h.factChecker.GetCheck(id)
				data, _ := json.Marshal(check)
				fmt.Fprintf(w, "event: complete\ndata: %s\n\n", data)
				flusher.Flush()
				return
			}

		case <-r.Context().Done():
			return
		}
	}
}
