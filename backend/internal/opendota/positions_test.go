package opendota

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func positionMatchFixture(id int64) map[string]any {
	players := make([]map[string]any, 0, 10)
	for team := range 2 {
		for position := 1; position <= 5; position++ {
			players = append(players, map[string]any{
				"hero_id": team*5 + position, "player_slot": team*128 + position - 1,
				"position_est": position,
			})
		}
	}
	return map[string]any{
		"match_id": id, "leagueid": 1, "start_time": 1700000000 + id,
		"radiant_win": true, "players": players,
	}
}

func positionTestServer(t *testing.T, matches map[int64]map[string]any, calls *atomic.Int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/heroes":
			heroes := make([]Hero, 10)
			for i := range heroes {
				heroes[i] = Hero{ID: i + 1, LocalizedName: fmt.Sprintf("Hero %d", i+1)}
			}
			_ = json.NewEncoder(w).Encode(heroes)
		case "/proMatches":
			calls.Add(1)
			_ = json.NewEncoder(w).Encode([]map[string]int64{
				{"match_id": 101}, {"match_id": 102}, {"match_id": 103}, {"match_id": 104}, {"match_id": 101},
			})
		default:
			var id int64
			if _, err := fmt.Sscanf(r.URL.Path, "/matches/%d", &id); err != nil {
				http.NotFound(w, r)
				return
			}
			match, ok := matches[id]
			if !ok {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			_ = json.NewEncoder(w).Encode(match)
		}
	}))
}

func TestHeroPositionStatsAggregatesEstimatedPositions(t *testing.T) {
	first := positionMatchFixture(101)
	second := positionMatchFixture(102)
	second["radiant_win"] = false
	players := second["players"].([]map[string]any)
	players[2]["position_est"], players[3]["position_est"] = 4, 3
	withoutPositions := positionMatchFixture(103)
	delete(withoutPositions["players"].([]map[string]any)[0], "position_est")
	var calls atomic.Int32
	server := positionTestServer(t, map[int64]map[string]any{101: first, 102: second, 103: withoutPositions}, &calls)
	defer server.Close()
	client := NewClient()
	client.baseURL, client.httpClient = server.URL, server.Client()

	snapshot, err := client.HeroPositionStats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Sample.MatchesRequested != 4 || snapshot.Sample.MatchesIncluded != 2 ||
		snapshot.Sample.MatchesUnavailable != 1 || snapshot.Sample.MatchesWithoutPositions != 1 {
		t.Fatalf("amostra inesperada: %+v", snapshot.Sample)
	}
	if !slices.Equal(snapshot.Sample.MatchIDs, []int64{101, 102}) {
		t.Fatalf("partidas inesperadas: %v", snapshot.Sample.MatchIDs)
	}
	if snapshot.Sample.From.IsZero() || snapshot.Sample.To.Before(snapshot.Sample.From) || snapshot.Sample.FetchedAt.IsZero() {
		t.Fatalf("período da amostra inválido: %+v", snapshot.Sample)
	}
	found := map[[2]int]PositionHeroStat{}
	totals := map[int]int{}
	for _, stat := range snapshot.Stats {
		found[[2]int{stat.HeroID, stat.Position}] = stat
		totals[stat.Position] += stat.Picks
	}
	for position := 1; position <= 5; position++ {
		if totals[position] != 4 {
			t.Errorf("posição %d: %d escolhas; esperado: 4", position, totals[position])
		}
	}
	for _, key := range [][2]int{{3, 3}, {3, 4}} {
		stat, ok := found[key]
		if !ok || stat.Picks != 1 || stat.LocalizedName != "Hero 3" {
			t.Fatalf("estatística inesperada para %v: %+v", key, stat)
		}
	}
	if found[[2]int{3, 3}].Wins != 1 || found[[2]int{3, 4}].Wins != 0 {
		t.Fatal("as vitórias devem ser calculadas por posição e pelo time vencedor")
	}

	snapshot.Stats[0].Picks = 999
	snapshot.Sample.MatchIDs[0] = 999
	cached, err := client.HeroPositionStats(context.Background())
	if err != nil || cached.Stats[0].Picks == 999 || cached.Sample.MatchIDs[0] == 999 || calls.Load() != 1 {
		t.Fatalf("cache alterado ou não reutilizado: %+v; erro: %v; consultas: %d", cached, err, calls.Load())
	}
}

func TestHeroPositionStatsConcurrentCacheAndExpiration(t *testing.T) {
	var calls atomic.Int32
	server := positionTestServer(t, map[int64]map[string]any{101: positionMatchFixture(101)}, &calls)
	defer server.Close()
	client := NewClient()
	client.baseURL, client.httpClient = server.URL, server.Client()
	var wg sync.WaitGroup
	start := make(chan struct{})
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			snapshot, err := client.HeroPositionStats(context.Background())
			if err != nil || snapshot.Sample.MatchesIncluded != 1 {
				t.Errorf("amostra inesperada: %+v; erro: %v", snapshot.Sample, err)
			}
		}()
	}
	close(start)
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("consultas: %d; esperado: 1", calls.Load())
	}
	client.positionsMu.Lock()
	client.positionsExpiresAt = time.Now().Add(-time.Second)
	client.positionsMu.Unlock()
	if _, err := client.HeroPositionStats(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("consultas após expirar: %d; esperado: 2", calls.Load())
	}
}

func TestHeroPositionStatsRejectsIncompleteLineups(t *testing.T) {
	for _, scenario := range []string{"missing_position", "duplicate_position", "invalid_slot", "missing_winner", "wrong_match"} {
		t.Run(scenario, func(t *testing.T) {
			match := positionMatchFixture(101)
			players := match["players"].([]map[string]any)
			switch scenario {
			case "missing_position":
				delete(players[0], "position_est")
				players[0]["lane_role"] = 1
			case "duplicate_position":
				players[0]["position_est"] = 2
			case "invalid_slot":
				players[0]["player_slot"] = 99
			case "missing_winner":
				delete(match, "radiant_win")
			case "wrong_match":
				match["match_id"] = 999
			}
			var calls atomic.Int32
			server := positionTestServer(t, map[int64]map[string]any{101: match}, &calls)
			defer server.Close()
			client := NewClient()
			client.baseURL, client.httpClient = server.URL, server.Client()
			for range 2 {
				if _, err := client.HeroPositionStats(context.Background()); err == nil {
					t.Fatal("amostras sem partidas utilizáveis devem gerar erro")
				}
			}
			if calls.Load() != 2 {
				t.Fatal("uma coleta sem dados válidos não deve preencher o cache")
			}
		})
	}
}

func TestHeroPositionStatsCapsSampleAndHonorsCancellation(t *testing.T) {
	var matchCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/proMatches":
			ids := make([]map[string]int64, 30)
			for i := range ids {
				ids[i] = map[string]int64{"match_id": int64(i + 1)}
			}
			_ = json.NewEncoder(w).Encode(ids)
		case "/heroes":
			heroes := make([]Hero, 10)
			for i := range heroes {
				heroes[i] = Hero{ID: i + 1, LocalizedName: fmt.Sprintf("Hero %d", i+1)}
			}
			_ = json.NewEncoder(w).Encode(heroes)
		default:
			var id int64
			if _, err := fmt.Sscanf(r.URL.Path, "/matches/%d", &id); err != nil {
				http.NotFound(w, r)
				return
			}
			matchCalls.Add(1)
			_ = json.NewEncoder(w).Encode(positionMatchFixture(id))
		}
	}))
	defer server.Close()
	client := NewClient()
	client.baseURL, client.httpClient = server.URL, server.Client()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.HeroPositionStats(ctx); err == nil || matchCalls.Load() != 0 {
		t.Fatal("consulta cancelada não deve acessar as partidas")
	}
	snapshot, err := client.HeroPositionStats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if matchCalls.Load() != 20 || snapshot.Sample.MatchLimit != 20 || snapshot.Sample.MatchesRequested != 20 || snapshot.Sample.MatchesIncluded != 20 {
		t.Fatalf("amostra deveria respeitar o limite de 20 partidas: consultas=%d; %+v", matchCalls.Load(), snapshot.Sample)
	}
}
