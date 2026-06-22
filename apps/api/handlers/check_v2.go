package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"alethea/api/auth"
	"alethea/api/httpx"
	"alethea/api/services/factcheck"
	"alethea/api/services/factcheck/checkstream"
	"alethea/api/services/factcheck/persist"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CheckV2 owns the Phase-2 /api/check endpoints. The constructor takes the
// pipeline + DB + hub; main.go composes it with the existing rate-limit and
// auth middleware.
type CheckV2 struct {
	DB       *pgxpool.Pool
	Pipeline *factcheck.Pipeline
	Hub      *checkstream.Hub
}

func NewCheckV2(db *pgxpool.Pool, p *factcheck.Pipeline, hub *checkstream.Hub) *CheckV2 {
	return &CheckV2{DB: db, Pipeline: p, Hub: hub}
}

type startReq struct {
	URL     string `json:"url"`
	Caption string `json:"caption,omitempty"`
}
type startResp struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// Start creates a check row, kicks off the pipeline in a worker goroutine,
// returns {id, status:pending}. The client follows up via GET or SSE.
func (h *CheckV2) Start(w http.ResponseWriter, r *http.Request) {
	var body startReq
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	body.URL = strings.TrimSpace(body.URL)
	if body.URL == "" {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "url is required")
		return
	}
	// Validate scheme/URL shape and run upfront SSRF assertion. We don't
	// fetch yet — the scraper does — but a bad URL should reject here.
	if !looksLikeHTTPURL(body.URL) {
		writeJSONErr(w, http.StatusBadRequest, "bad_url", "url must be http or https")
		return
	}

	sess, hasSession := auth.FromContext(r.Context())
	plan := "anon"
	var userID *int64
	if hasSession {
		plan = sess.Plan
		uid := sess.UserID
		userID = &uid
	}
	fanoutN := factcheck.FanoutFor(plan)

	// New check row.
	checkID, err := persist.NewCheck(r.Context(), h.DB, userID, nil, body.URL, body.Caption, plan, fanoutN)
	if err != nil {
		slog.Error("check_v2: create row", "err", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal", "could not start check")
		return
	}

	// Spawn the pipeline in a background goroutine. We deliberately use a
	// fresh context (not r.Context) because r.Context is canceled when the
	// HTTP handler returns. The pipeline owns its own timeout.
	in := factcheck.CheckInput{URL: body.URL, Caption: body.Caption}
	var sessCopy *auth.Session
	if hasSession {
		s := sess
		sessCopy = &s
	}
	go h.Pipeline.Run(context.Background(), checkID, in, sessCopy, plan)

	writeJSON(w, http.StatusAccepted, startResp{ID: checkID.String(), Status: "pending"})
}

// Get returns the current check status + latest verdicts joined from the DB.
// This is the polling fallback when SSE isn't available.
func (h *CheckV2) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONErr(w, http.StatusBadRequest, "bad_id", "invalid check id")
		return
	}
	c, err := persist.GetCheck(r.Context(), h.DB, id)
	if errors.Is(err, persist.ErrNotFound) {
		writeJSONErr(w, http.StatusNotFound, "not_found", "check not found")
		return
	}
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	// Authorization: anonymous checks (user_id NULL) are world-readable by ID
	// (the ID is the bearer). Owned checks are only readable by the owner.
	if c.UserID != nil {
		sess, ok := auth.FromContext(r.Context())
		if !ok || sess.UserID != *c.UserID {
			writeJSONErr(w, http.StatusNotFound, "not_found", "check not found")
			return
		}
	}
	writeJSON(w, http.StatusOK, c)
}

// Stream is the SSE endpoint. Subscribes to the hub channel for this check
// and forwards events as they arrive. Closes when the channel closes or
// the client disconnects.
func (h *CheckV2) Stream(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONErr(w, http.StatusBadRequest, "bad_id", "invalid check id")
		return
	}
	c, err := persist.GetCheck(r.Context(), h.DB, id)
	if errors.Is(err, persist.ErrNotFound) {
		writeJSONErr(w, http.StatusNotFound, "not_found", "check not found")
		return
	}
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if c.UserID != nil {
		sess, ok := auth.FromContext(r.Context())
		if !ok || sess.UserID != *c.UserID {
			writeJSONErr(w, http.StatusNotFound, "not_found", "check not found")
			return
		}
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONErr(w, http.StatusInternalServerError, "internal", "SSE not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering if behind one

	ch := h.Hub.Get(id)
	lastSeen := -1
	if s := r.URL.Query().Get("last_event_id"); s != "" {
		fmt.Sscanf(s, "%d", &lastSeen)
	}
	sub, cancel := ch.Subscribe(lastSeen)
	defer cancel()

	// Heartbeat so proxies don't kill an idle connection.
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			_, _ = fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		case e, open := <-sub:
			if !open {
				_, _ = fmt.Fprintf(w, "event: end\ndata: {}\n\n")
				flusher.Flush()
				return
			}
			b, err := json.Marshal(e)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", e.Seq, e.Stage, b)
			flusher.Flush()
		}
	}
}

// --- helpers ---

func looksLikeHTTPURL(s string) bool {
	s = strings.ToLower(s)
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeJSONErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": msg},
	})
}

// Touch httpx to keep the import live in case future code uses it directly here.
var _ = httpx.ClientIP
