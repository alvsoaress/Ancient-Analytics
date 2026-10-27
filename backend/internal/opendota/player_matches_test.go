package opendota

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPlayerMatchesForwardsPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if r.Method != "GET" || r.URL.Path != "/players/42/matches" || query.Get("limit") != "3" || query.Get("offset") != "2" || query.Get("sort") != "match_id" || query.Get("significant") != "0" {
			t.Errorf("consulta inesperada: %s %s", r.Method, r.URL)
		}
		if r.Header.Get("User-Agent") != "AncientAnalytics/1.0" {
			t.Error("consulta deve identificar a aplicação")
		}
		fmt.Fprint(w, `[{"match_id":9025128045,"hero_id":53,"player_slot":0,"radiant_win":true,"kills":0,"deaths":2,"assists":16,"duration":2375,"start_time":1790890789}]`)
	}))
	defer server.Close()
	client := NewClient()
	client.baseURL, client.httpClient = server.URL, server.Client()
	matches, err := client.PlayerMatches(context.Background(), 42, 3, 2)
	if err != nil || len(matches) != 1 {
		t.Fatalf("partidas: %+v; erro: %v", matches, err)
	}
	match := matches[0]
	if match.MatchID != 9025128045 || match.HeroID == nil || *match.HeroID != 53 || match.Kills == nil || *match.Kills != 0 || match.RadiantWin == nil || !*match.RadiantWin {
		t.Fatalf("partida inesperada: %+v", match)
	}
}

func TestPlayerMatchesUnavailableAndMalformedResponses(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
	}{
		{"http_403", 403, `{}`}, {"http_404", 404, `{}`}, {"http_429", 429, `{}`}, {"http_503", 503, `{}`},
		{"json_invalido", 200, `[`}, {"null", 200, `null`}, {"objeto", 200, `{}`},
		{"sem_id", 200, `[{}]`}, {"id_negativo", 200, `[{"match_id":-1}]`},
		{"duplicadas", 200, `[{"match_id":1},{"match_id":1}]`},
		{"abates_negativos", 200, `[{"match_id":1,"kills":-1}]`},
		{"slot_invalido", 200, `[{"match_id":1,"player_slot":256}]`},
		{"mais_que_limite", 200, `[{"match_id":3},{"match_id":2},{"match_id":1}]`},
		{"resposta_grande", 200, strings.Repeat(" ", (2<<20)+1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
				fmt.Fprint(w, test.body)
			}))
			defer server.Close()
			client := NewClient()
			client.baseURL, client.httpClient = server.URL, server.Client()
			if _, err := client.PlayerMatches(context.Background(), 42, 2, 0); err == nil {
				t.Fatal("resposta inválida deveria retornar erro")
			}
		})
	}
}

func TestPlayerMatchesEmptyAndMissingFields(t *testing.T) {
	for _, body := range []string{`[]`, `[{"match_id":1}]`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
		client := NewClient()
		client.baseURL, client.httpClient = server.URL, server.Client()
		matches, err := client.PlayerMatches(context.Background(), 42, 2, 0)
		server.Close()
		if err != nil || matches == nil {
			t.Fatalf("lista válida não deveria falhar: %v", err)
		}
		if len(matches) > 0 && (matches[0].RadiantWin != nil || matches[0].PlayerSlot != nil || matches[0].Kills != nil) {
			t.Fatal("dados ausentes devem permanecer indisponíveis")
		}
	}
}

func TestPlayerMatchesRejectsInvalidInputAndCancellation(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); fmt.Fprint(w, `[]`) }))
	defer server.Close()
	client := NewClient()
	client.baseURL, client.httpClient = server.URL, server.Client()
	for _, test := range []struct {
		id            uint32
		limit, offset int
	}{{0, 2, 0}, {42, 0, 0}, {42, 102, 0}, {42, 2, -1}, {42, 2, 2147483647}} {
		if _, err := client.PlayerMatches(context.Background(), test.id, test.limit, test.offset); err == nil {
			t.Fatalf("entrada inválida aceita: %+v", test)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.PlayerMatches(ctx, 42, 2, 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelamento não preservado: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatal("entrada inválida ou cancelamento não deve consultar a fonte")
	}
}
