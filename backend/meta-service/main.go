package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func iniciarHeartbeatPush() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	chave := "service:meta:8082:status"
	ticker := time.NewTicker(5 * time.Second)

	rdb.Set(ctx, chave, "vivo", 10*time.Second)
	log.Println("[Heartbeat] Iniciando envio de pings para o Redis...")

	go func() {
		for range ticker.C {
			err := rdb.Set(ctx, chave, "vivo", 10*time.Second).Err()
			if err != nil {
				log.Println("⚠️ Erro ao enviar ping para o Redis:", err)
			}
		}
	}()
}

func main() {
	iniciarHeartbeatPush()

	http.HandleFunc("/api/meta/dashboard", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"meta_status": "Dados do patch atual carregados com sucesso!", "winrate": 55.4}`)
	})

	log.Println("Meta Service do Ancient Analytics rodando na porta 8082...")
	log.Fatal(http.ListenAndServe(":8082", nil))
}
