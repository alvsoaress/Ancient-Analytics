package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alvsoaress/ancient-analytics-backend/internal/meta"
	"github.com/alvsoaress/ancient-analytics-backend/internal/opendota"
)

func (f fakeMetaService) HeroesByPosition(ctx context.Context, minPicks int, sortBy string, position meta.Position) (meta.PositionResult, error) {
	if f.receivedMinPicks != nil {
		*f.receivedMinPicks = minPicks
	}
	return meta.PositionResult{}, f.err
}

func (fakeMetaStatsClient) HeroPositionStats(context.Context) (opendota.PositionSnapshot, error) {
	return opendota.PositionSnapshot{}, errors.New("consulta por posição inesperada")
}

type positionAPIClient struct{}

func (positionAPIClient) HeroStats(context.Context) ([]opendota.HeroStat, error) {
	return []opendota.HeroStat{{ID: 99, LocalizedName: "Overall", ProPick: 1000, ProWin: 999}}, nil
}

func (positionAPIClient) HeroPositionStats(context.Context) (opendota.PositionSnapshot, error) {
	return opendota.PositionSnapshot{
		Stats: []opendota.PositionHeroStat{
			{HeroID: 1, LocalizedName: "Hero 1", Position: 1, Picks: 3, Wins: 2},
			{HeroID: 4, LocalizedName: "Hero 4", Position: 1, Picks: 1, Wins: 1},
			{HeroID: 1, LocalizedName: "Hero 1", Position: 2, Picks: 2, Wins: 1},
			{HeroID: 3, LocalizedName: "Hero 3", Position: 3, Picks: 2, Wins: 1},
			{HeroID: 2, LocalizedName: "Hero 2", Position: 4, Picks: 2, Wins: 2},
			{HeroID: 2, LocalizedName: "Hero 2", Position: 5, Picks: 2, Wins: 0},
		},
		Sample: opendota.PositionSample{MatchLimit: 20, MatchesRequested: 2, MatchesIncluded: 2, MatchIDs: []int64{101, 102}},
	}, nil
}

func TestMetaHeroesPositionQueries(t *testing.T) {
	handler := NewHandler(fakeHeroesClient{}, meta.NewService(positionAPIClient{}), fakePlayersService{}, fakeMatchesService{}, fakeBuildsService{})
	for _, value := range []string{"carry", "mid", "offlane", "support4", "support5"} {
		t.Run(value, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/meta/heroes?position="+value+"&minPicks=2&sortBy=winRate", nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status: %d; corpo: %s", response.Code, response.Body.String())
			}
			var body struct {
				Scope          string                  `json:"scope"`
				Position       string                  `json:"position"`
				PositionSource string                  `json:"positionSource"`
				Heroes         []meta.HeroPerformance  `json:"heroes"`
				Sample         opendota.PositionSample `json:"sample"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Scope != "professional" || body.Position != value || body.PositionSource != "opendota.position_est" || body.Sample.MatchesIncluded != 2 {
				t.Fatalf("metadados inesperados: %+v", body)
			}
			if len(body.Heroes) != 1 || body.Heroes[0].HeroID == 99 || body.Heroes[0].Picks < 2 || body.Heroes[0].WinRate == nil {
				t.Fatalf("o endpoint deve usar a amostra específica da posição: %+v", body.Heroes)
			}
			if value == "support4" && *body.Heroes[0].WinRate != 100 || value == "support5" && *body.Heroes[0].WinRate != 0 {
				t.Fatal("suportes 4 e 5 devem ter estatísticas independentes")
			}
		})
	}
}

func TestMetaHeroesRejectsInvalidPosition(t *testing.T) {
	for _, query := range []string{"position=", "position=jungle", "position=3", "position=offlane&position=mid"} {
		t.Run(query, func(t *testing.T) {
			called := -1
			handler := NewHandler(fakeHeroesClient{}, fakeMetaService{receivedMinPicks: &called}, fakePlayersService{}, fakeMatchesService{}, fakeBuildsService{})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/meta/heroes?"+query, nil))
			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusBadRequest || body.Error.Code != "INVALID_POSITION" || called != -1 {
				t.Fatalf("posição inválida deveria ser rejeitada antes da consulta: %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestMetaHeroesPositionSourceUnavailable(t *testing.T) {
	handler := NewHandler(fakeHeroesClient{}, fakeMetaService{err: errors.New("sem partidas com posições")}, fakePlayersService{}, fakeMatchesService{}, fakeBuildsService{})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/meta/heroes?position=offlane", nil))
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusBadGateway || body.Error.Code != "POSITION_DATA_UNAVAILABLE" {
		t.Fatalf("falha da fonte não deve virar agregado geral ou lista de zeros: %d %s", response.Code, response.Body.String())
	}
}
