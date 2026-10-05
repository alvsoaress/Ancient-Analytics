package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

type HeroStat struct {
	ID            int      `json:"id"`
	LocalizedName string   `json:"localized_name"`
	PrimaryAttr   string   `json:"primary_attr"`
	AttackType    string   `json:"attack_type"`
	Roles         []string `json:"roles"`
	ProPick       int      `json:"pro_pick"`
	ProBan        int      `json:"pro_ban"`
	ProWin        int      `json:"pro_win"`
	ImmortalPick  int      `json:"8_pick"`
	ImmortalWin   int      `json:"8_win"`
	AncientPick   int      `json:"6_pick"`
	AncientWin    int      `json:"6_win"`
}

// ItemStat mapeia os itens do jogo
type ItemStat struct {
	ID            int      `json:"id"`
	Cost          int      `json:"cost"`
	LocalizedName string   `json:"localized_name"`
	Hint          []string `json:"hint"`
	Recipe        int      `json:"recipe"`
}

// Caches em memória
var (
	heroesCache    []HeroStat
	lastHeroesTime time.Time
	heroesMutex    sync.RWMutex

	itemsCache    map[string]ItemStat
	lastItemsTime time.Time
	itemsMutex    sync.RWMutex

	cacheDuration = 10 * time.Minute
)

func fetchHeroes() ([]HeroStat, error) {
	heroesMutex.RLock()
	if time.Since(lastHeroesTime) < cacheDuration && heroesCache != nil {
		defer heroesMutex.RUnlock()
		return heroesCache, nil
	}
	heroesMutex.RUnlock()

	heroesMutex.Lock()
	defer heroesMutex.Unlock()

	if time.Since(lastHeroesTime) < cacheDuration && heroesCache != nil {
		return heroesCache, nil
	}

	log.Println("Buscando Heróis na API do OpenDota...")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.opendota.com/api/heroStats")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var heroes []HeroStat
	if err := json.NewDecoder(resp.Body).Decode(&heroes); err != nil {
		return nil, err
	}

	heroesCache = heroes
	lastHeroesTime = time.Now()
	return heroesCache, nil
}

func fetchItems() (map[string]ItemStat, error) {
	itemsMutex.RLock()
	if time.Since(lastItemsTime) < cacheDuration && itemsCache != nil {
		defer itemsMutex.RUnlock()
		return itemsCache, nil
	}
	itemsMutex.RUnlock()

	itemsMutex.Lock()
	defer itemsMutex.Unlock()

	if time.Since(lastItemsTime) < cacheDuration && itemsCache != nil {
		return itemsCache, nil
	}

	log.Println("Buscando Itens na API do OpenDota...")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.opendota.com/api/constants/items")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var items map[string]ItemStat
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}

	itemsCache = items
	lastItemsTime = time.Now()
	return itemsCache, nil
}

func iniciarHeartbeatPush() {
	ticker := time.NewTicker(5 * time.Second)
	gatewayURL := "http://localhost:8080/internal/heartbeat/meta"

	log.Println("[Heartbeat] Iniciando envio de pings HTTP para o Gateway...")
	client := &http.Client{Timeout: 2 * time.Second}

	go func() {
		for range ticker.C {
			req, err := http.NewRequest(http.MethodPost, gatewayURL, nil)
			if err == nil {
				resp, err := client.Do(req)
				if err == nil {
					resp.Body.Close()
				}
			}
		}
	}()
}

func main() {
	iniciarHeartbeatPush()

	// Rota para todos os HERÓIS
	http.HandleFunc("/api/meta/heroes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		stats, err := fetchHeroes()
		if err != nil {
			http.Error(w, `{"erro": "Falha ao buscar heróis"}`, http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(stats)
	})

	// Rota para todos os ITENS
	http.HandleFunc("/api/meta/items", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		items, err := fetchItems()
		if err != nil {
			http.Error(w, `{"erro": "Falha ao buscar itens"}`, http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(items)
	})

	log.Println("Meta Service do Ancient Analytics rodando na porta 8082...")
	log.Fatal(http.ListenAndServe(":8082", nil))
}
