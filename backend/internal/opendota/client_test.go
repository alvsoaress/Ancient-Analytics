package opendota

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHeroes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/api/heroes" {
				t.Errorf("requisição inesperada: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[
				{"id": 1, "localized_name": "Anti-Mage"}
			]`))
		},
	))
	defer server.Close()

	client := NewClient()
	client.baseURL = server.URL + "/api"
	client.httpClient = server.Client()

	heroes, err := client.Heroes(context.Background())
	if err != nil {
		t.Fatalf("erro ao buscar heróis: %v", err)
	}

	if len(heroes) != 1 {
		t.Fatalf("quantidade de heróis: %d; esperado: 1", len(heroes))
	}

	if heroes[0].ID != 1 {
		t.Errorf("ID recebido: %d; esperado: 1", heroes[0].ID)
	}

	if heroes[0].LocalizedName != "Anti-Mage" {
		t.Errorf(
			"nome recebido: %q; esperado: Anti-Mage",
			heroes[0].LocalizedName,
		)
	}
}
