package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alvsoaress/ancient-analytics-backend/internal/matches"
)

type fakeMatchesService struct {
	result     matches.Details
	err        error
	receivedID *int64
}

func (f fakeMatchesService) Details(ctx context.Context, id int64) (matches.Details, error) {
	if f.receivedID != nil {
		*f.receivedID = id
	}
	return f.result, f.err
}

func TestMatchDetailsRoute(t *testing.T) {
	winner, team := "dire", "radiant"
	won := false
	kills := 0
	received := int64(0)
	service := fakeMatchesService{receivedID: &received, result: matches.Details{MatchID: 9025128045, Winner: &winner, Players: []matches.Player{{Team: &team, Won: &won, Kills: &kills}}}}
	handler := NewHandler(fakeHeroesClient{}, fakeMetaService{}, fakePlayersService{}, service)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/matches/9025128045", nil))
	if response.Code != 200 || response.Header().Get("Content-Type") != "application/json" || received != 9025128045 {
		t.Fatalf("resposta inesperada: %d %s", response.Code, response.Body.String())
	}
	var body matches.Details
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.MatchID != 9025128045 || body.Winner == nil || *body.Winner != "dire" || len(body.Players) != 1 || body.Players[0].Won == nil || *body.Players[0].Won || body.Players[0].Kills == nil || *body.Players[0].Kills != 0 {
		t.Fatalf("contrato inesperado: %+v", body)
	}
	for _, field := range []string{"accountId", "heroId", "name", "durationSeconds", "startTime", "radiantScore", "direScore", "goldPerMinute", "xpPerMinute", "heroDamage"} {
		if !strings.Contains(response.Body.String(), `"`+field+`":null`) {
			t.Errorf("campo ausente deve ser null: %s", field)
		}
	}
}

func TestMatchDetailsRejectsInvalidIDBeforeCallingService(t *testing.T) {
	for _, id := range []string{"0", "-1", "abc", "1.5", "9007199254740992", "999999999999999999999"} {
		t.Run(id, func(t *testing.T) {
			received := int64(0)
			handler := NewHandler(fakeHeroesClient{}, fakeMetaService{}, fakePlayersService{}, fakeMatchesService{receivedID: &received})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/matches/"+id, nil))
			if response.Code != 400 || received != 0 || !strings.Contains(response.Body.String(), `"code":"INVALID_MATCH_ID"`) {
				t.Fatalf("ID inválido aceito: %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestMatchDetailsErrorResponses(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"indisponivel", matches.ErrNotFound, 404, "MATCH_NOT_FOUND"},
		{"falha_na_fonte", errors.New("detalhe interno da conexão"), 502, "UPSTREAM_UNAVAILABLE"},
		{"id_invalido", matches.ErrInvalidMatchID, 400, "INVALID_MATCH_ID"},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := NewHandler(fakeHeroesClient{}, fakeMetaService{}, fakePlayersService{}, fakeMatchesService{err: test.err})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/matches/1", nil))
			var body struct {
				Error struct{ Code, Message string } `json:"error"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if response.Code != test.status || body.Error.Code != test.code || body.Error.Message == "" || strings.Contains(response.Body.String(), "detalhe interno") {
				t.Fatalf("erro inesperado: %d %s", response.Code, response.Body.String())
			}
		})
	}
}
