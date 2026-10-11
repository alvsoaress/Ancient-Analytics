package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type HeroStat struct {
	ID            int      `json:"id"`
	LocalizedName string   `json:"localized_name"`
	PrimaryAttr   string   `json:"primary_attr"`
	AttackType    string   `json:"attack_type"`
	Roles         []string `json:"roles"`
	ProPick       int      `json:"pro_pick"`
	ProBan        int      `json:"pro_ban"`
	ProWin        int      `json:"pro_win"`
	ImmortalPick  int      `json:"8_pick"`
	ImmortalWin   int      `json:"8_win"`
	AncientPick   int      `json:"6_pick"`
	AncientWin    int      `json:"6_win"`
}

type ItemStat struct {
	ID            int      `json:"id"`
	Cost          int      `json:"cost"`
	LocalizedName string   `json:"localized_name"`
	Hint          []string `json:"hint"`
	Recipe        int      `json:"recipe"`
}

type Config struct {
	OpenDotaURL string
	CacheTTL    time.Duration
	HTTPClient  *http.Client
}

type metaService struct {
	baseURL string
	ttl     time.Duration
	client  *http.Client

	heroesMu        sync.RWMutex
	heroes          []HeroStat
	heroesUpdatedAt time.Time
	itemsMu         sync.RWMutex
	items           map[string]ItemStat
	itemsUpdatedAt  time.Time
}

func NewHandler(config Config) http.Handler {
	if config.OpenDotaURL == "" {
		config.OpenDotaURL = "https://api.opendota.com/api"
	}
	if config.CacheTTL <= 0 {
		config.CacheTTL = 10 * time.Minute
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	service := &metaService{baseURL: strings.TrimRight(config.OpenDotaURL, "/"), ttl: config.CacheTTL, client: config.HTTPClient}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /api/meta/heroes", func(w http.ResponseWriter, r *http.Request) {
		heroes, err := service.fetchHeroes(r.Context())
		if err != nil {
			writeError(w, http.StatusBadGateway, "UPSTREAM_UNAVAILABLE", "could not retrieve hero data")
			return
		}
		writeJSON(w, http.StatusOK, heroes)
	})
	mux.HandleFunc("GET /api/meta/items", func(w http.ResponseWriter, r *http.Request) {
		items, err := service.fetchItems(r.Context())
		if err != nil {
			writeError(w, http.StatusBadGateway, "UPSTREAM_UNAVAILABLE", "could not retrieve item data")
			return
		}
		writeJSON(w, http.StatusOK, items)
	})
	return mux
}

func (s *metaService) fetchHeroes(ctx context.Context) ([]HeroStat, error) {
	s.heroesMu.RLock()
	if s.heroes != nil && time.Since(s.heroesUpdatedAt) < s.ttl {
		defer s.heroesMu.RUnlock()
		return s.heroes, nil
	}
	s.heroesMu.RUnlock()

	s.heroesMu.Lock()
	defer s.heroesMu.Unlock()
	if s.heroes != nil && time.Since(s.heroesUpdatedAt) < s.ttl {
		return s.heroes, nil
	}
	var heroes []HeroStat
	if err := s.getJSON(ctx, "/heroStats", &heroes); err != nil {
		return nil, err
	}
	s.heroes, s.heroesUpdatedAt = heroes, time.Now()
	return s.heroes, nil
}

func (s *metaService) fetchItems(ctx context.Context) (map[string]ItemStat, error) {
	s.itemsMu.RLock()
	if s.items != nil && time.Since(s.itemsUpdatedAt) < s.ttl {
		defer s.itemsMu.RUnlock()
		return s.items, nil
	}
	s.itemsMu.RUnlock()

	s.itemsMu.Lock()
	defer s.itemsMu.Unlock()
	if s.items != nil && time.Since(s.itemsUpdatedAt) < s.ttl {
		return s.items, nil
	}
	var items map[string]ItemStat
	if err := s.getJSON(ctx, "/constants/items", &items); err != nil {
		return nil, err
	}
	s.items, s.itemsUpdatedAt = items, time.Now()
	return s.items, nil
}

func (s *metaService) getJSON(ctx context.Context, path string, output any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("opendota returned status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(output)
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
		port = "8082"
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	server := &http.Server{Addr: ":" + port, Handler: NewHandler(Config{OpenDotaURL: os.Getenv("OPENDOTA_URL")}), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		logger.Info("meta service started", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("meta service stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("meta service shutdown failed", "error", err)
	}
}
