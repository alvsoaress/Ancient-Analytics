package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Registry struct {
	mu       sync.RWMutex
	services map[string]time.Time
}

func NewRegistry() *Registry {
	return &Registry{
		services: make(map[string]time.Time),
	}
}

func (r *Registry) UpdateHeartbeat(serviceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services[serviceID] = time.Now()
}

func (r *Registry) IsAlive(serviceID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	lastPing, exists := r.services[serviceID]
	if !exists {
		return false
	}

	return time.Since(lastPing) < 10*time.Second
}

func criaProxy(destinoHost string) *httputil.ReverseProxy {
	destino, _ := url.Parse(destinoHost)
	return httputil.NewSingleHostReverseProxy(destino)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	registry := NewRegistry()

	draftProxy := criaProxy("http://localhost:8081")
	metaProxy := criaProxy("http://localhost:8082")

	mux := http.NewServeMux()

	mux.HandleFunc("POST /internal/heartbeat/{service}", func(w http.ResponseWriter, r *http.Request) {
		serviceID := r.PathValue("service")
		registry.UpdateHeartbeat(serviceID)
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		status := map[string]bool{
			"Meta Service":  registry.IsAlive("meta"),
			"Draft Service": registry.IsAlive("draft"),
		}
		_ = json.NewEncoder(w).Encode(status)
	})

	mux.HandleFunc("/api/meta/", func(w http.ResponseWriter, r *http.Request) {
		if !registry.IsAlive("meta") {
			http.Error(w, "Meta Service indisponível no momento", http.StatusServiceUnavailable)
			return
		}
		metaProxy.ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/draft/", func(w http.ResponseWriter, r *http.Request) {
		if !registry.IsAlive("draft") {
			http.Error(w, "Draft Service indisponível no momento", http.StatusServiceUnavailable)
			return
		}
		draftProxy.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("Gateway do Ancient Analytics iniciado (Memória Nativa)", "addr", server.Addr)
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("falha critica no servidor", "erro", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()
	<-ctx.Done()

	logger.Info("Sinal de encerramento recebido. Iniciando graceful shutdown...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("falha durante o shutdown", "erro", err)
	} else {
		logger.Info("Gateway encerrado com segurança.")
	}
}
