package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alvsoaress/ancient-analytics-backend/internal/meta"
)

type fakeMetaService struct {
	heroes []meta.HeroPerformance
	err    error
}

func (f fakeMetaService) Heroes(
	ctx context.Context,
	minPicks int,
) ([]meta.HeroPerformance, error) {
	return f.heroes, f.err
}

func TestMetaHeroes(t *testing.T) {
	rate := 37.5

	service := fakeMetaService{
		heroes: []meta.HeroPerformance{
			{
				HeroID:  1,
				Name:    "Anti-Mage",
				Picks:   8,
				Wins:    3,
				WinRate: &rate,
			},
		},
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/meta/heroes",
		nil,
	)
	response := httptest.NewRecorder()

	handler := NewHandler(fakeHeroesClient{}, service)
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf(
			"status recebido: %d; esperado: 200",
			response.Code,
		)
	}

	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatal("a resposta deveria ter Content-Type application/json")
	}

	var body struct {
		Scope  string                 `json:"scope"`
		Heroes []meta.HeroPerformance `json:"heroes"`
	}

	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("resposta JSON inválida: %v", err)
	}

	if body.Scope != "professional" {
		t.Errorf(
			"recorte recebido: %q; esperado: professional",
			body.Scope,
		)
	}

	if len(body.Heroes) != 1 {
		t.Fatalf(
			"quantidade de resultados: %d; esperado: 1",
			len(body.Heroes),
		)
	}

	hero := body.Heroes[0]

	if hero.HeroID != 1 || hero.Name != "Anti-Mage" {
		t.Errorf("identificação inesperada: %+v", hero)
	}

	if hero.Picks != 8 || hero.Wins != 3 {
		t.Errorf("estatísticas inesperadas: %+v", hero)
	}

	if hero.WinRate == nil {
		t.Fatal("esperada taxa de vitória disponível")
	}

	if *hero.WinRate != 37.5 {
		t.Errorf(
			"taxa recebida: %v; esperado: 37.5",
			*hero.WinRate,
		)
	}
}

func TestMetaHeroesReturnsBadGateway(t *testing.T) {
	service := fakeMetaService{
		err: errors.New("falha ao consultar o provedor"),
	}

	handler := NewHandler(fakeHeroesClient{}, service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/meta/heroes",
		nil,
	)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadGateway {
		t.Fatalf(
			"status recebido: %d; esperado: 502",
			response.Code,
		)
	}

	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatal("a resposta deveria ter Content-Type application/json")
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("resposta JSON inválida: %v", err)
	}

	if body.Error.Code != "UPSTREAM_UNAVAILABLE" {
		t.Errorf("código de erro inesperado: %q", body.Error.Code)
	}

	if body.Error.Message != "Não foi possível consultar as estatísticas de meta." {
		t.Errorf("mensagem inesperada: %q", body.Error.Message)
	}
}

func TestMetaHeroesWithoutPicksReturnsNullRate(t *testing.T) {
	service := fakeMetaService{
		heroes: []meta.HeroPerformance{
			{
				HeroID:  1,
				Name:    "Anti-Mage",
				Picks:   0,
				Wins:    0,
				WinRate: nil,
			},
		},
	}

	handler := NewHandler(fakeHeroesClient{}, service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/meta/heroes",
		nil,
	)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf(
			"status recebido: %d; esperado: 200",
			response.Code,
		)
	}

	var body struct {
		Scope  string `json:"scope"`
		Heroes []struct {
			WinRate json.RawMessage `json:"winRate"`
		} `json:"heroes"`
	}

	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("resposta JSON inválida: %v", err)
	}

	if body.Scope != "professional" {
		t.Errorf("recorte inesperado: %q", body.Scope)
	}

	if len(body.Heroes) != 1 {
		t.Fatalf(
			"quantidade de resultados: %d; esperado: 1",
			len(body.Heroes),
		)
	}

	if string(body.Heroes[0].WinRate) != "null" {
		t.Fatalf(
			"esperado winRate: null; recebido: %s",
			body.Heroes[0].WinRate,
		)
	}
}
