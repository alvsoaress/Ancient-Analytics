package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alvsoaress/ancient-analytics-backend/internal/players"
)

func (f fakePlayersService) Matches(context.Context, uint32, int, int) (players.MatchHistory, error) {
	return players.MatchHistory{}, nil
}

type fakeHistoryService struct {
	fakePlayersService
	result               players.MatchHistory
	err                  error
	id                   uint32
	limit, offset, calls int
}

func (f *fakeHistoryService) Matches(ctx context.Context, id uint32, limit, offset int) (players.MatchHistory, error) {
	f.id, f.limit, f.offset = id, limit, offset
	f.calls++
	return f.result, f.err
}

func TestPlayerMatchesRouteAndPaginationDefaults(t *testing.T) {
	for _, test := range []struct {
		query         string
		limit, offset int
	}{{"", 20, 0}, {"?limit=2&offset=4", 2, 4}, {"?limit=100&offset=0", 100, 0}} {
		next := test.offset + test.limit
		service := &fakeHistoryService{result: players.MatchHistory{AccountID: 42, Matches: []players.Match{{MatchID: 9025128045}}, Pagination: players.Pagination{Limit: test.limit, Offset: test.offset, Count: 1, HasMore: true, NextOffset: &next}}}
		handler := NewHandler(fakeHeroesClient{}, fakeMetaService{}, service)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/players/42/matches"+test.query, nil))
		if response.Code != 200 || service.id != 42 || service.limit != test.limit || service.offset != test.offset || response.Header().Get("Content-Type") != "application/json" {
			t.Fatalf("resposta inesperada: %d %s; consulta: %+v", response.Code, response.Body.String(), service)
		}
		var body players.MatchHistory
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.AccountID != 42 || len(body.Matches) != 1 || body.Matches[0].MatchID != 9025128045 || !body.Pagination.HasMore || body.Pagination.NextOffset == nil || *body.Pagination.NextOffset != next {
			t.Fatalf("contrato inesperado: %+v", body)
		}
		if !strings.Contains(response.Body.String(), `"won":null`) || !strings.Contains(response.Body.String(), `"kills":null`) {
			t.Fatal("dados indisponíveis devem ser null")
		}
	}
}

func TestPlayerMatchesRejectsInvalidQueryBeforeCallingService(t *testing.T) {
	for _, test := range []struct{ path, code string }{
		{"/0/matches", "INVALID_ACCOUNT_ID"}, {"/abc/matches", "INVALID_ACCOUNT_ID"}, {"/4294967296/matches", "INVALID_ACCOUNT_ID"},
		{"/42/matches?limit=0", "INVALID_LIMIT"}, {"/42/matches?limit=101", "INVALID_LIMIT"}, {"/42/matches?limit=-1", "INVALID_LIMIT"},
		{"/42/matches?limit=", "INVALID_LIMIT"}, {"/42/matches?limit=abc", "INVALID_LIMIT"}, {"/42/matches?limit=1.5", "INVALID_LIMIT"},
		{"/42/matches?limit=1&limit=2", "INVALID_LIMIT"}, {"/42/matches?limit=999999999999999999999", "INVALID_LIMIT"},
		{"/42/matches?offset=-1", "INVALID_OFFSET"}, {"/42/matches?offset=", "INVALID_OFFSET"}, {"/42/matches?offset=abc", "INVALID_OFFSET"},
		{"/42/matches?offset=1.5", "INVALID_OFFSET"}, {"/42/matches?offset=1&offset=2", "INVALID_OFFSET"}, {"/42/matches?offset=2147483647", "INVALID_OFFSET"},
		{"/42/matches?limit=%zz", "INVALID_PAGINATION"},
	} {
		t.Run(test.path, func(t *testing.T) {
			service := &fakeHistoryService{}
			response := httptest.NewRecorder()
			NewHandler(fakeHeroesClient{}, fakeMetaService{}, service).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/players"+test.path, nil))
			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if response.Code != 400 || body.Error.Code != test.code || service.calls != 0 {
				t.Fatalf("entrada inválida aceita: %d %s; consultas: %d", response.Code, response.Body.String(), service.calls)
			}
		})
	}
}

func TestPlayerMatchesEmptyHistoryAndSourceFailure(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status int
	}{{"vazia", nil, 200}, {"fonte_falhou", errors.New("detalhe interno do servidor"), 502}} {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeHistoryService{result: players.MatchHistory{AccountID: 42, Matches: []players.Match{}, Pagination: players.Pagination{Limit: 20}}, err: test.err}
			response := httptest.NewRecorder()
			NewHandler(fakeHeroesClient{}, fakeMetaService{}, service).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/players/42/matches", nil))
			if response.Code != test.status {
				t.Fatalf("status inesperado: %d %s", response.Code, response.Body.String())
			}
			if test.err == nil {
				if !strings.Contains(response.Body.String(), `"matches":[]`) || !strings.Contains(response.Body.String(), `"hasMore":false`) || !strings.Contains(response.Body.String(), `"nextOffset":null`) {
					t.Fatal("histórico vazio deve encerrar a paginação com lista vazia")
				}
			} else if !strings.Contains(response.Body.String(), `"code":"UPSTREAM_UNAVAILABLE"`) || strings.Contains(response.Body.String(), "detalhe interno") {
				t.Fatal("falha deve ter mensagem genérica")
			}
		})
	}
}
