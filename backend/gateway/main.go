package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type Config struct {
	MetaServiceURL    string
	DraftServiceURL   string
	UserServiceURL    string
	DependencyTimeout time.Duration
}

type dependency struct {
	name string
	url  *url.URL
}

func NewHandler(config Config) (http.Handler, error) {
	if config.MetaServiceURL == "" {
		config.MetaServiceURL = "http://meta-service:8082"
	}
	if config.DraftServiceURL == "" {
		config.DraftServiceURL = "http://draft-service:8081"
	}
	if config.UserServiceURL == "" {
		config.UserServiceURL = "http://user-service:8083"
	}
	if config.DependencyTimeout <= 0 {
		config.DependencyTimeout = 3 * time.Second
	}
	meta, err := parseDependency("meta", config.MetaServiceURL)
	if err != nil {
		return nil, err
	}
	draft, err := parseDependency("draft", config.DraftServiceURL)
	if err != nil {
		return nil, err
	}
	user, err := parseDependency("user", config.UserServiceURL)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 2 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: time.Second,
	}

	mux := http.NewServeMux()
	mux.Handle("/api/meta/", reverseProxy(meta.url, transport))
	mux.Handle("/api/draft/", reverseProxy(draft.url, transport))
	mux.Handle("/api/users/", reverseProxy(user.url, transport))
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		statuses := make(chan struct {
			name string
			ok   bool
		}, 3)
		for _, service := range []dependency{meta, draft, user} {
			go func(dep dependency) {
				statuses <- struct {
					name string
					ok   bool
				}{dep.name, isReady(r.Context(), dep.url, config.DependencyTimeout)}
			}(service)
		}
		result := map[string]bool{}
		ready := true
		for range 3 {
			status := <-statuses
			result[status.name] = status.ok
			ready = ready && status.ok
		}
		if !ready {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "unavailable", "dependencies": result})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ready", "dependencies": result})
	})
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{
			"Meta Service":  isReady(r.Context(), meta.url, config.DependencyTimeout),
			"Draft Service": isReady(r.Context(), draft.url, config.DependencyTimeout),
		})
	})

	return requestID(mux), nil
}

func parseDependency(name, rawURL string) (dependency, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return dependency{}, fmt.Errorf("invalid %s service URL", name)
	}
	return dependency{name: name, url: parsed}, nil
}

func reverseProxy(target *url.URL, transport http.RoundTripper) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
		writeError(w, http.StatusBadGateway, "UPSTREAM_UNAVAILABLE", "service is temporarily unavailable")
	}
	return proxy
}

func isReady(parent context.Context, service *url.URL, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	readyURL := *service
	readyURL.Path = strings.TrimRight(readyURL.Path, "/") + "/readyz"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, readyURL.String(), nil)
	if err != nil {
		return false
	}
	resp, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = newRequestID()
			r.Header.Set("X-Request-ID", id)
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r)
	})
}

func newRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
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
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	handler, err := NewHandler(Config{MetaServiceURL: os.Getenv("META_SERVICE_URL"), DraftServiceURL: os.Getenv("DRAFT_SERVICE_URL"), UserServiceURL: os.Getenv("USER_SERVICE_URL")})
	if err != nil {
		logger.Error("invalid gateway configuration", "error", err)
		os.Exit(1)
	}
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}
	server := &http.Server{Addr: ":" + port, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		logger.Info("gateway started", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("gateway stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("gateway shutdown failed", "error", err)
	}
}
