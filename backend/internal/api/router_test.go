package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alvsoaress/ancient-analytics-backend/internal/opendota"
)

type fakeHeroesClient struct {
	err error
}

func (f fakeHeroesClient) Heroes(
	ctx context.Context,
) ([]opendota.Hero, error) {
	if f.err != nil {
		return nil, f.err
	}

	return []opendota.Hero{
		{ID: 1, LocalizedName: "Anti-Mage"},
	}, nil
}

func TestHealth(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/health",
		nil,
	)
	response := httptest.NewRecorder()

	client := opendota.NewClient()
	handler := NewHandler(client, fakeMetaService{}, fakePlayersService{}, fakeMatchesService{}, fakeBuildsService{})
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status recebido: %d; esperado: 200", response.Code)
	}

	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatal("a resposta deveria ter Content-Type application/json")
	}

	var body struct {
		Status string `json:"status"`
	}

	err := json.Unmarshal(response.Body.Bytes(), &body)
	if err != nil {
		t.Fatalf("resposta JSON inválida: %v", err)
	}

	if body.Status != "ok" {
		t.Fatalf("status recebido: %q; esperado: ok", body.Status)
	}
}

func TestHeroes(t *testing.T) {
	handler := NewHandler(fakeHeroesClient{}, fakeMetaService{}, fakePlayersService{}, fakeMatchesService{}, fakeBuildsService{})

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/heroes",
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

	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatal("a resposta deveria ter Content-Type application/json")
	}

	var heroes []opendota.Hero

	if err := json.Unmarshal(response.Body.Bytes(), &heroes); err != nil {
		t.Fatalf("resposta JSON inválida: %v", err)
	}

	if len(heroes) != 1 {
		t.Fatalf("quantidade de heróis: %d; esperado: 1", len(heroes))
	}

	if heroes[0].ID != 1 || heroes[0].LocalizedName != "Anti-Mage" {
		t.Fatalf("herói inesperado: %+v", heroes[0])
	}
}

func TestHeroesReturnsBadGateway(t *testing.T) {
	client := fakeHeroesClient{
		err: errors.New("falha de conexão com o provedor"),
	}
	handler := NewHandler(client, fakeMetaService{}, fakePlayersService{}, fakeMatchesService{}, fakeBuildsService{})

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/heroes",
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

	if body.Error.Message != "Não foi possível consultar os heróis." {
		t.Errorf("mensagem inesperada: %q", body.Error.Message)
	}
}
