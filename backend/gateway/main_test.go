package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGatewayRoutesMetaAndReportsDependencyReadiness(t *testing.T) {
	meta := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/readyz":
			w.WriteHeader(http.StatusOK)
		case "/api/meta/heroes":
			_, _ = w.Write([]byte(`[{"id":1}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer meta.Close()
	draft := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/readyz" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		http.NotFound(w, r)
	}))
	defer draft.Close()

	handler, err := NewHandler(Config{MetaServiceURL: meta.URL, DraftServiceURL: draft.URL})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	proxyResponse := httptest.NewRecorder()
	handler.ServeHTTP(proxyResponse, httptest.NewRequest(http.MethodGet, "/api/meta/heroes", nil))
	if proxyResponse.Code != http.StatusOK {
		t.Fatalf("proxy status = %d, want %d", proxyResponse.Code, http.StatusOK)
	}

	readyResponse := httptest.NewRecorder()
	handler.ServeHTTP(readyResponse, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if readyResponse.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready status = %d, want %d", readyResponse.Code, http.StatusServiceUnavailable)
	}

	statusResponse := httptest.NewRecorder()
	handler.ServeHTTP(statusResponse, httptest.NewRequest(http.MethodGet, "/api/status", nil))
	if statusResponse.Code != http.StatusOK {
		t.Fatalf("status endpoint = %d, want %d", statusResponse.Code, http.StatusOK)
	}
}
