package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/health",
		nil,
	)
	response := httptest.NewRecorder()

	handler := NewHandler()
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
