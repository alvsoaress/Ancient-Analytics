package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func criaProxy(destinoHost string) *httputil.ReverseProxy {
	destino, _ := url.Parse(destinoHost)
	return httputil.NewSingleHostReverseProxy(destino)
}

func main() {
	// Conecta ao Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	draftProxy := criaProxy("http://localhost:8081")
	metaProxy := criaProxy("http://localhost:8082")

	mux := http.NewServeMux()

	// 1. O painel geral agora lê diretamente do Redis
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// .Val() retorna a quantidade de chaves encontradas. Se > 0, o serviço está vivo.
		metaAlive := rdb.Exists(ctx, "service:meta:8082:status").Val() > 0
		draftAlive := rdb.Exists(ctx, "service:draft:8081:status").Val() > 0

		status := map[string]bool{
			"Meta Service":  metaAlive,
			"Draft Service": draftAlive,
		}
		json.NewEncoder(w).Encode(status)
	})

	// 2. Protege as rotas consultando a validade da chave antes de rotear
	mux.HandleFunc("/api/meta/", func(w http.ResponseWriter, r *http.Request) {
		if rdb.Exists(ctx, "service:meta:8082:status").Val() == 0 {
			http.Error(w, "Meta Service indisponível no momento", http.StatusServiceUnavailable)
			return
		}
		metaProxy.ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/draft/", func(w http.ResponseWriter, r *http.Request) {
		if rdb.Exists(ctx, "service:draft:8081:status").Val() == 0 {
			http.Error(w, "Draft Service indisponível no momento", http.StatusServiceUnavailable)
			return
		}
		draftProxy.ServeHTTP(w, r)
	})

	log.Println("Gateway Central orquestrando via Redis na porta 8080...")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
