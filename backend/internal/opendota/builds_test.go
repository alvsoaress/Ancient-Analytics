package opendota

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testBuildClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)
	c := NewClient()
	c.baseURL = server.URL
	c.httpClient = server.Client()
	return c
}

const buildMatchJSON = `{"match_id":30,"leagueid":7,"start_time":1000,"duration":2000,"patch":50,"radiant_win":true,"players":[{"hero_id":62,"player_slot":128,"position_est":4,"purchase_log":[{"key":"blink","time":900}],"ability_upgrades_arr":[5118]}]}`

func TestHeroBuildsCacheAndCopy(t *testing.T) {
	var calls atomic.Int32
	c := testBuildClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch r.URL.Path {
		case "/heroes/62/matches":
			fmt.Fprint(w, `[{"match_id":30},{"match_id":30},{"match_id":0}]`)
		case "/matches/30":
			fmt.Fprint(w, buildMatchJSON)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(404)
		}
	})
	const requests = 8
	var wg sync.WaitGroup
	for range requests {
		wg.Go(func() {
			r, err := c.HeroBuilds(context.Background(), 62)
			if err != nil {
				t.Error(err)
				return
			}
			if len(r.Matches) != 1 || r.Matches[0].Won || r.Matches[0].Position != 4 || r.MatchesRequested != 1 {
				t.Errorf("snapshot: %+v", r)
			}
		})
	}
	wg.Wait()
	if calls.Load() != 2 {
		t.Fatalf("calls: %d", calls.Load())
	}
	r, _ := c.HeroBuilds(context.Background(), 62)
	r.Matches[0].Purchases[0].Key = "modified"
	r.Matches[0].AbilityUpgrades[0] = 0
	*r.Matches[0].Patch = 0
	again, _ := c.HeroBuilds(context.Background(), 62)
	if again.Matches[0].Purchases[0].Key != "blink" || again.Matches[0].AbilityUpgrades[0] != 5118 || *again.Matches[0].Patch != 50 {
		t.Fatal("caller changed cache")
	}
	c.buildSamplesMu.Lock()
	entry := c.buildSamplesCache[62]
	entry.expiresAt = time.Now().Add(-time.Second)
	c.buildSamplesCache[62] = entry
	c.buildSamplesMu.Unlock()
	if _, err := c.HeroBuilds(context.Background(), 62); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 4 {
		t.Fatalf("cache did not expire: %d", calls.Load())
	}
}
func TestHeroBuildsReportsUnusableMatches(t *testing.T) {
	c := testBuildClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/heroes/62/matches":
			fmt.Fprint(w, `[{"match_id":30},{"match_id":31},{"match_id":32}]`)
		case "/matches/30":
			fmt.Fprint(w, buildMatchJSON)
		case "/matches/31":
			fmt.Fprint(w, strings.ReplaceAll(strings.ReplaceAll(buildMatchJSON, `"match_id":30`, `"match_id":31`), `"purchase_log":[{"key":"blink","time":900}]`, `"purchase_log":null`))
		default:
			w.WriteHeader(429)
		}
	})
	r, err := c.HeroBuilds(context.Background(), 62)
	if err != nil {
		t.Fatal(err)
	}
	if r.MatchesRequested != 3 || r.MatchesUnavailable != 1 || r.MatchesWithoutPurchaseLog != 1 || len(r.Matches) != 1 {
		t.Fatalf("counts: %+v", r)
	}
}
func TestHeroBuildsUnavailableIsNotEmptySuccess(t *testing.T) {
	for _, body := range []string{`null`, `{}`, `broken`, `[{"match_id":-1}]`} {
		t.Run(body, func(t *testing.T) {
			c := testBuildClient(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) })
			if _, err := c.HeroBuilds(context.Background(), 62); err == nil {
				t.Fatal("invalid list accepted")
			}
		})
	}
	c := testBuildClient(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/heroes/") {
			fmt.Fprint(w, `[{"match_id":30}]`)
		} else {
			w.WriteHeader(503)
		}
	})
	if _, err := c.HeroBuilds(context.Background(), 62); err == nil {
		t.Fatal("all upstream failures became empty success")
	}
}
func TestHeroBuildsRejectsInvalidMatchData(t *testing.T) {
	for name, body := range map[string]string{"mismatch": strings.ReplaceAll(buildMatchJSON, `"match_id":30`, `"match_id":31`), "no_professional_league": strings.ReplaceAll(buildMatchJSON, `"leagueid":7`, `"leagueid":0`), "out_of_range_time": strings.ReplaceAll(buildMatchJSON, `"start_time":1000`, `"start_time":9223372036854775807`),
		"negative_time": strings.ReplaceAll(buildMatchJSON, `"start_time":1000`, `"start_time":-1`), "missing_slot": strings.ReplaceAll(buildMatchJSON, `"player_slot":128,`, ""), "wrong_hero": strings.ReplaceAll(buildMatchJSON, `"hero_id":62`, `"hero_id":1`), "negative_ability": strings.ReplaceAll(buildMatchJSON, `[5118]`, `[-1]`), "null_purchase": strings.ReplaceAll(buildMatchJSON, `[{"key":"blink","time":900}]`, `[null]`)} {
		t.Run(name, func(t *testing.T) {
			c := testBuildClient(t, func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/heroes/") {
					fmt.Fprint(w, `[{"match_id":30}]`)
				} else {
					fmt.Fprint(w, body)
				}
			})
			r, err := c.HeroBuilds(context.Background(), 62)
			if err == nil || len(r.Matches) != 0 {
				t.Fatalf("invalid match accepted: %+v %v", r, err)
			}
		})
	}
}
func TestHeroBuildsEmptyListAndMissingReplay(t *testing.T) {
	for _, noReplay := range []bool{false, true} {
		c := testBuildClient(t, func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/heroes/") {
				if noReplay {
					fmt.Fprint(w, `[{"match_id":30}]`)
				} else {
					fmt.Fprint(w, `[]`)
				}
			} else {
				fmt.Fprint(w, strings.ReplaceAll(buildMatchJSON, `"purchase_log":[{"key":"blink","time":900}]`, `"purchase_log":null`))
			}
		})
		r, err := c.HeroBuilds(context.Background(), 62)
		if err != nil || r.Matches == nil || len(r.Matches) != 0 {
			t.Fatalf("no data: %+v %v", r, err)
		}
	}
}
func TestBuildCatalogCachesAndClones(t *testing.T) {
	var calls atomic.Int32
	c := testBuildClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch r.URL.Path {
		case "/constants/items":
			fmt.Fprint(w, `{"blink":{"id":1,"dname":"Blink Dagger","cost":2250,"img":"/apps/dota2/images/dota_react/items/blink.png"}}`)
		case "/constants/abilities":
			fmt.Fprint(w, `{"bounty_hunter_track":{"dname":"Track"}}`)
		case "/constants/ability_ids":
			fmt.Fprint(w, `{"5118":"bounty_hunter_track"}`)
		case "/constants/hero_abilities":
			fmt.Fprint(w, `{"npc_dota_hero_bounty_hunter":{"abilities":["bounty_hunter_track"],"talents":[{"name":"special_bonus_test","level":1},{"name":"rubick_hidden3","level":5}]}}`)
		default:
			w.WriteHeader(404)
		}
	})
	first, err := c.BuildCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	h := first.HeroAbilities["npc_dota_hero_bounty_hunter"]
	h.Abilities[0] = "modified"
	h.Talents[0].Name = "modified"
	delete(first.Items, "blink")
	delete(first.AbilityIDs, 5118)
	again, err := c.BuildCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 4 || len(again.Items) != 1 || len(again.AbilityIDs) != 1 || again.HeroAbilities["npc_dota_hero_bounty_hunter"].Abilities[0] != "bounty_hunter_track" || again.HeroAbilities["npc_dota_hero_bounty_hunter"].Talents[0].Name != "special_bonus_test" {
		t.Fatal("catalog was fetched again or cache mutated")
	}
}
func TestBuildCatalogRejectsInvalidAndDoesNotCacheErrors(t *testing.T) {
	var calls atomic.Int32
	c := testBuildClient(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); fmt.Fprint(w, `null`) })
	for range 2 {
		if _, err := c.BuildCatalog(context.Background()); err == nil {
			t.Fatal("null catalog accepted")
		}
	}
	if calls.Load() != 2 {
		t.Fatal("error cached")
	}
}
func TestHeroBuildsCancellationAndInvalidID(t *testing.T) {
	c := testBuildClient(t, func(w http.ResponseWriter, r *http.Request) { t.Error("unexpected network call") })
	for _, id := range []int{0, -1, 2147483648} {
		if _, err := c.HeroBuilds(context.Background(), id); err == nil {
			t.Fatal("invalid hero accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.HeroBuilds(ctx, 62); err == nil {
		t.Fatal("cancel ignored")
	}
	if _, err := c.BuildCatalog(ctx); err == nil {
		t.Fatal("cancel ignored")
	}
}

func TestHeroBuildsBoundsSampleAndConcurrency(t *testing.T) {
	var active, peak, total atomic.Int32
	c := testBuildClient(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/heroes/") {
			fmt.Fprint(w, "[")
			for i := range 30 {
				if i > 0 {
					fmt.Fprint(w, ",")
				}
				fmt.Fprintf(w, `{"match_id":%d}`, 30+i)
			}
			fmt.Fprint(w, "]")
			return
		}
		n := active.Add(1)
		for old := peak.Load(); n > old; old = peak.Load() {
			if peak.CompareAndSwap(old, n) {
				break
			}
		}
		defer active.Add(-1)
		total.Add(1)
		time.Sleep(time.Millisecond)
		id := strings.TrimPrefix(r.URL.Path, "/matches/")
		fmt.Fprint(w, strings.ReplaceAll(buildMatchJSON, `"match_id":30`, `"match_id":`+id))
	})
	r, err := c.HeroBuilds(context.Background(), 62)
	if err != nil {
		t.Fatal(err)
	}
	if total.Load() != 20 || r.MatchesRequested != 20 || len(r.Matches) != 20 || peak.Load() > 4 || peak.Load() < 2 {
		t.Fatalf("sample/concurrency: requested %d peak %d calls %d", r.MatchesRequested, peak.Load(), total.Load())
	}
}
func TestBuildJSONRejectsOversizedResponse(t *testing.T) {
	c := testBuildClient(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, strings.Repeat(" ", (16<<20)+1)) })
	var result any
	if err := c.getBuildJSON(context.Background(), "/large", &result); err == nil || !strings.Contains(err.Error(), "excede") {
		t.Fatalf("oversized response accepted: %v", err)
	}
}

func TestBuildCatalogExpandsCombinedAbilityIdentifiers(t *testing.T) {
	c := testBuildClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/constants/items":
			fmt.Fprint(w, `{"blink":{"id":1,"cost":2250}}`)
		case "/constants/abilities":
			fmt.Fprint(w, `{"templar_assassin_hidden_gates":{"dname":"Hidden Gates"}}`)
		case "/constants/ability_ids":
			fmt.Fprint(w, `{"3060,1617":"templar_assassin_hidden_gates"}`)
		case "/constants/hero_abilities":
			fmt.Fprint(w, `{"npc_dota_hero_templar_assassin":{"abilities":["templar_assassin_hidden_gates"]}}`)
		}
	})
	r, err := c.BuildCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if r.AbilityIDs[3060] != "templar_assassin_hidden_gates" || r.AbilityIDs[1617] != "templar_assassin_hidden_gates" {
		t.Fatalf("combined identifiers: %+v", r.AbilityIDs)
	}
}

func TestBuildHeroAbilitiesFlattensAlternativeAbilityNames(t *testing.T) {
	var h BuildHeroAbilities
	err := json.Unmarshal([]byte(`{"abilities":["monkey_king_mischief",["monkey_king_untransform","monkey_king_transfiguration"]],"talents":[]}`), &h)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Abilities) != 3 || h.Abilities[2] != "monkey_king_transfiguration" {
		t.Fatalf("ability names: %+v", h.Abilities)
	}
}

func TestHeroBuildsKeepsFinalInventoryAndDuration(t *testing.T) {
	body := strings.Replace(buildMatchJSON, `"position_est":4,`, `"position_est":4,"item_0":137,"item_1":63,"item_2":116,"item_3":1,"item_4":0,"item_5":137,`, 1)
	c := testBuildClient(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/heroes/") {
			fmt.Fprint(w, `[{"match_id":30}]`)
		} else {
			fmt.Fprint(w, body)
		}
	})
	result, err := c.HeroBuilds(context.Background(), 62)
	if err != nil {
		t.Fatal(err)
	}
	if result.Matches[0].Duration != 2000 || len(result.Matches[0].FinalItems) != 6 || result.Matches[0].FinalItems[0] != 137 || result.Matches[0].FinalItems[4] != 0 {
		t.Fatalf("inventário perdido: %+v", result.Matches[0])
	}
	result.Matches[0].FinalItems[0] = 0
	cached, err := c.HeroBuilds(context.Background(), 62)
	if err != nil || cached.Matches[0].FinalItems[0] != 137 {
		t.Fatal("inventário em cache foi alterado pelo consumidor")
	}
}
