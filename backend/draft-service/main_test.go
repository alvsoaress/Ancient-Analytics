package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDraftSessionsCreateAndRejectDuplicateHeroes(t *testing.T) {
	handler := NewHandler()

	create := httptest.NewRequest(http.MethodPost, "/api/draft/sessions", nil)
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, create)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", createResponse.Code, http.StatusCreated)
	}

	var session DraftSession
	if err := json.NewDecoder(createResponse.Body).Decode(&session); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	if session.ID == "" {
		t.Fatal("expected a generated session ID")
	}

	payload := []byte(`{"type":"PICK","team":"RADIANT","heroId":1}`)
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/api/draft/sessions/"+session.ID+"/events", bytes.NewReader(payload)))
	if first.Code != http.StatusCreated {
		t.Fatalf("first event status = %d, want %d", first.Code, http.StatusCreated)
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/api/draft/sessions/"+session.ID+"/events", bytes.NewReader(payload)))
	if second.Code != http.StatusConflict {
		t.Fatalf("duplicate event status = %d, want %d", second.Code, http.StatusConflict)
	}
}
