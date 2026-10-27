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

func TestMatchFetchesPublicDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/matches/9025128045" || r.Header.Get("User-Agent") != "AncientAnalytics/1.0" {
			t.Errorf("requisição inesperada: %s %s", r.Method, r.URL)
		}
		fmt.Fprint(w, `{"match_id":9025128045,"duration":2375,"radiant_win":false,"radiant_score":26,"dire_score":43,"players":[{"account_id":42,"personaname":"Player","player_slot":128,"hero_id":53,"kills":0,"deaths":2,"assists":16,"gold_per_min":500,"xp_per_min":600,"hero_damage":10000,"tower_damage":2000,"hero_healing":0}]}`)
	}))
	defer server.Close()
	client := NewClient()
	client.baseURL, client.httpClient = server.URL, server.Client()
	data, err := client.Match(context.Background(), 9025128045)
	if err != nil || data.MatchID != 9025128045 || data.RadiantWin == nil || *data.RadiantWin || data.Duration == nil || *data.Duration != 2375 || len(data.Players) != 1 {
		t.Fatalf("partida inesperada: %+v; erro: %v", data, err)
	}
	p := data.Players[0]
	if p.AccountID == nil || *p.AccountID != 42 || p.PersonaName == nil || *p.PersonaName != "Player" || p.PlayerSlot == nil || *p.PlayerSlot != 128 || p.Kills == nil || *p.Kills != 0 || p.GoldPerMinute == nil || *p.GoldPerMinute != 500 || p.HeroDamage == nil || *p.HeroDamage != 10000 {
		t.Fatalf("jogador inesperado: %+v", p)
	}
}

func TestMatchRejectsInvalidSourceResponses(t *testing.T) {
	for _, test := range []struct {
		name     string
		status   int
		body     string
		notFound bool
	}{
		{"http_404", 404, `{}`, true}, {"http_403", 403, `{}`, false}, {"http_429", 429, `{}`, false}, {"http_503", 503, `{}`, false},
		{"json_invalido", 200, `{`, false}, {"json_null", 200, `null`, false}, {"id_incorreto", 200, `{"match_id":2}`, false}, {"sem_id", 200, `{}`, false},
		{"duracao_negativa", 200, `{"match_id":1,"duration":-1}`, false}, {"placar_negativo", 200, `{"match_id":1,"dire_score":-1}`, false},
		{"estatistica_negativa", 200, `{"match_id":1,"players":[{"kills":-1}]}`, false},
		{"slot_invalido", 200, `{"match_id":1,"players":[{"player_slot":256}]}`, false},
		{"slot_repetido", 200, `{"match_id":1,"players":[{"player_slot":0},{"player_slot":0}]}`, false},
		{"conta_invalida", 200, `{"match_id":1,"players":[{"account_id":4294967296}]}`, false},
		{"jogador_null", 200, `{"match_id":1,"players":[null]}`, false},
		{"resposta_grande", 200, strings.Repeat(" ", (16<<20)+1), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(test.status); fmt.Fprint(w, test.body) }))
			defer server.Close()
			client := NewClient()
			client.baseURL, client.httpClient = server.URL, server.Client()
			_, err := client.Match(context.Background(), 1)
			if err == nil || errors.Is(err, ErrMatchNotFound) != test.notFound {
				t.Fatalf("erro inesperado: %v; esperava indisponível: %t", err, test.notFound)
			}
		})
	}
}

func TestMatchAllowsPartialDataAndNonstandardSlots(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"match_id":1,"players":[{"player_slot":6},{"player_slot":134},{}]}`)
	}))
	defer server.Close()
	client := NewClient()
	client.baseURL, client.httpClient = server.URL, server.Client()
	data, err := client.Match(context.Background(), 1)
	if err != nil || len(data.Players) != 3 || data.RadiantWin != nil || data.Players[2].AccountID != nil || data.Players[2].PlayerSlot != nil {
		t.Fatalf("dados parciais não preservados: %+v; erro: %v", data, err)
	}
}

func TestMatchRejectsInvalidIDAndPreservesCancellation(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); fmt.Fprint(w, `{"match_id":1}`) }))
	defer server.Close()
	client := NewClient()
	client.baseURL, client.httpClient = server.URL, server.Client()
	for _, id := range []int64{0, -1, 9007199254740992} {
		if _, err := client.Match(context.Background(), id); err == nil {
			t.Fatal("identificador inválido aceito")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Match(ctx, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelamento não preservado: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatal("consulta inválida não deve acessar a fonte")
	}
}
