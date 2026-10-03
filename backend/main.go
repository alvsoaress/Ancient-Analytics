package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"status": "Backend Go rodando!"}`)
	})
	
	fmt.Println("Servidor rodando na porta 8080...")
	http.ListenAndServe(":8080", nil)
}