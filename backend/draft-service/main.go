package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type DraftEvent struct {
	Type      string    `json:"type"`
	Team      string    `json:"team"`
	HeroID    int       `json:"heroId"`
	CreatedAt time.Time `json:"createdAt"`
}

type DraftSession struct {
	ID        string       `json:"id"`
	Events    []DraftEvent `json:"events"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
}

type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]DraftSession
}

func newSessionStore() *sessionStore {
	return &sessionStore{sessions: make(map[string]DraftSession)}
}

func (s *sessionStore) create() (DraftSession, error) {
	id, err := newID()
	if err != nil {
		return DraftSession{}, err
	}
	now := time.Now().UTC()
	session := DraftSession{ID: id, Events: []DraftEvent{}, CreatedAt: now, UpdatedAt: now}
	s.mu.Lock()
	s.sessions[id] = session
	s.mu.Unlock()
	return session, nil
}

func (s *sessionStore) get(id string) (DraftSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[id]
	return session, ok
}

func (s *sessionStore) addEvent(id string, event DraftEvent) (DraftSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return DraftSession{}, errSessionNotFound
	}
	for _, existing := range session.Events {
		if existing.HeroID == event.HeroID {
			return DraftSession{}, errHeroAlreadySelected
		}
	}
	event.CreatedAt = time.Now().UTC()
	session.Events = append(session.Events, event)
	session.UpdatedAt = event.CreatedAt
	s.sessions[id] = session
	return session, nil
}

var (
	errSessionNotFound     = errors.New("draft session not found")
	errHeroAlreadySelected = errors.New("hero already selected in this draft")
)

func NewHandler() http.Handler {
	store := newSessionStore()
	mux := http.NewServeMux()

	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("POST /api/draft/sessions", func(w http.ResponseWriter, _ *http.Request) {
		session, err := store.create()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "SESSION_CREATION_FAILED", "could not create draft session")
			return
		}
		writeJSON(w, http.StatusCreated, session)
	})
	mux.HandleFunc("GET /api/draft/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		session, ok := store.get(r.PathValue("id"))
		if !ok {
			writeError(w, http.StatusNotFound, "SESSION_NOT_FOUND", "draft session not found")
			return
		}
		writeJSON(w, http.StatusOK, session)
	})
	mux.HandleFunc("POST /api/draft/sessions/{id}/events", func(w http.ResponseWriter, r *http.Request) {
		var event DraftEvent
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&event); err != nil || !validEvent(event) {
			writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "type must be PICK or BAN, team must be RADIANT or DIRE, and heroId must be positive")
			return
		}
		session, err := store.addEvent(r.PathValue("id"), event)
		switch {
		case errors.Is(err, errSessionNotFound):
			writeError(w, http.StatusNotFound, "SESSION_NOT_FOUND", err.Error())
		case errors.Is(err, errHeroAlreadySelected):
			writeError(w, http.StatusConflict, "HERO_ALREADY_SELECTED", err.Error())
		case err != nil:
			writeError(w, http.StatusInternalServerError, "EVENT_CREATION_FAILED", "could not record draft event")
		default:
			writeJSON(w, http.StatusCreated, session)
		}
	})

	return mux
}

func validEvent(event DraftEvent) bool {
	return (event.Type == "PICK" || event.Type == "BAN") &&
		(event.Team == "RADIANT" || event.Team == "DIRE") && event.HeroID > 0
}

func newID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func main() {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8081"
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	server := &http.Server{Addr: ":" + port, Handler: NewHandler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		logger.Info("draft service started", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("draft service stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("draft service shutdown failed", "error", err)
	}
}
