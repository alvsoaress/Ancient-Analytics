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
	heroes           []meta.HeroPerformance
	err              error
	receivedMinPicks *int
}

func (f fakeMetaService) Heroes(
	ctx context.Context,
	minPicks int,
) ([]meta.HeroPerformance, error) {
	if f.receivedMinPicks != nil {
		*f.receivedMinPicks = minPicks
	}
	return f.heroes, f.err
}

func TestMetaHeroesPassesMinPicks(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want int
	}{
		{name: "sem_parametro", url: "/api/meta/heroes", want: 0},
		{name: "minimo_dez", url: "/api/meta/heroes?minPicks=10", want: 10},
		{name: "zero_explicito", url: "/api/meta/heroes?minPicks=0", want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			receivedMinPicks := -1
			service := fakeMetaService{
				heroes:           []meta.HeroPerformance{},
				receivedMinPicks: &receivedMinPicks,
			}
			handler := NewHandler(fakeHeroesClient{}, service)
			request := httptest.NewRequest(http.MethodGet, test.url, nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status recebido: %d; esperado: 200", response.Code)
			}
			if receivedMinPicks != test.want {
				t.Errorf("mínimo recebido pelo serviço: %d; esperado: %d", receivedMinPicks, test.want)
			}
		})
	}
}

func TestMetaHeroesRejectsInvalidMinPicks(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{name: "negativo", query: "minPicks=-1"},
		{name: "texto", query: "minPicks=abc"},
		{name: "decimal", query: "minPicks=1.5"},
		{name: "vazio", query: "minPicks="},
		{name: "sem_valor", query: "minPicks"},
		{name: "fora_do_limite_de_int", query: "minPicks=999999999999999999999999999999"},
		{name: "repetido", query: "minPicks=10&minPicks=20"},
		{name: "codificacao_invalida", query: "minPicks=%ZZ"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			receivedMinPicks := -1
			service := fakeMetaService{receivedMinPicks: &receivedMinPicks}
			handler := NewHandler(fakeHeroesClient{}, service)
			request := httptest.NewRequest(http.MethodGet, "/api/meta/heroes?"+test.query, nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status recebido: %d; esperado: 400", response.Code)
			}
			if response.Header().Get("Content-Type") != "application/json" {
				t.Error("a resposta deveria ter Content-Type application/json")
			}
			if receivedMinPicks != -1 {
				t.Error("o serviço não deveria ser chamado para um parâmetro inválido")
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
			if body.Error.Code != "INVALID_MIN_PICKS" || body.Error.Message != "minPicks deve ser um único número inteiro maior ou igual a zero." {
				t.Errorf("erro inesperado: %+v", body.Error)
			}
		})
	}
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
