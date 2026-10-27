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

type fakePlayersService struct {
	profile    players.Profile
	err        error
	receivedID *uint32
}

func (f fakePlayersService) Profile(ctx context.Context, id uint32) (players.Profile, error) {
	if f.receivedID != nil {
		*f.receivedID = id
	}
	return f.profile, f.err
}

func TestPlayerProfileRoute(t *testing.T) {
	name, steamID := "Player", "76561198000000000"
	avatar, profileURL := "https://avatars.steamstatic.com/avatar.jpg", "https://steamcommunity.com/profiles/76561198000000000/"
	rank := 80
	received := uint32(0)
	handler := NewHandler(fakeHeroesClient{}, fakeMetaService{}, fakePlayersService{
		profile: players.Profile{AccountID: 42, Name: &name, SteamID: &steamID, AvatarURL: &avatar, ProfileURL: &profileURL, RankTier: &rank}, receivedID: &received,
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/players/42", nil))
	if response.Code != http.StatusOK || received != 42 || response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("resposta inesperada: %d %s", response.Code, response.Body.String())
	}
	var body struct {
		AccountID       uint32          `json:"accountId"`
		Name            string          `json:"name"`
		SteamID         string          `json:"steamId"`
		AvatarURL       string          `json:"avatarUrl"`
		ProfileURL      string          `json:"profileUrl"`
		RankTier        int             `json:"rankTier"`
		LeaderboardRank json.RawMessage `json:"leaderboardRank"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.AccountID != 42 || body.Name != name || body.SteamID != steamID || body.AvatarURL != avatar || body.ProfileURL != profileURL || body.RankTier != 80 || string(body.LeaderboardRank) != "null" {
		t.Fatalf("contrato inesperado: %+v", body)
	}
}

func TestPlayerProfileUnavailableFieldsAreNull(t *testing.T) {
	handler := NewHandler(fakeHeroesClient{}, fakeMetaService{}, fakePlayersService{profile: players.Profile{AccountID: 42}})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/players/42", nil))
	var body map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"name", "steamId", "avatarUrl", "profileUrl", "rankTier", "leaderboardRank"} {
		if string(body[field]) != "null" {
			t.Errorf("campo %s: esperado null; recebido %s", field, body[field])
		}
	}
}

func TestPlayerProfileRejectsInvalidAccountID(t *testing.T) {
	for _, id := range []string{"0", "-1", "abc", "1.5", "4294967296", "76561198000000000"} {
		t.Run(id, func(t *testing.T) {
			received := uint32(0)
			handler := NewHandler(fakeHeroesClient{}, fakeMetaService{}, fakePlayersService{receivedID: &received})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/players/"+id, nil))
			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusBadRequest || body.Error.Code != "INVALID_ACCOUNT_ID" || received != 0 {
				t.Fatalf("identificador inválido não deveria consultar o serviço: %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestPlayerProfileErrorResponses(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"indisponivel", players.ErrNotFound, 404, "PLAYER_NOT_FOUND"},
		{"fonte_falhou", errors.New("detalhe interno da conexão"), 502, "UPSTREAM_UNAVAILABLE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := NewHandler(fakeHeroesClient{}, fakeMetaService{}, fakePlayersService{err: test.err})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/players/42", nil))
			var body struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if response.Code != test.status || body.Error.Code != test.code || body.Error.Message == "" || strings.Contains(response.Body.String(), "detalhe interno") {
				t.Fatalf("resposta de erro inesperada: %d %s", response.Code, response.Body.String())
			}
		})
	}
}
