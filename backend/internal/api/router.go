package api

import "net/http"

func NewHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", health)

	return mux
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
