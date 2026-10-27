package opendota

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"time"
)

const positionMatchLimit = 20

type PositionHeroStat struct {
	HeroID        int
	LocalizedName string
	Position      int
	Picks         int
	Wins          int
	// Somas de K/D/A apenas das escolhas em que a fonte informou os três valores.
	Kills    int
	Deaths   int
	Assists  int
	KDAPicks int
}

type PositionSample struct {
	MatchLimit              int       `json:"matchLimit"`
	MatchesRequested        int       `json:"matchesRequested"`
	MatchesIncluded         int       `json:"matchesIncluded"`
	MatchesUnavailable      int       `json:"matchesUnavailable"`
	MatchesWithoutPositions int       `json:"matchesWithoutPositions"`
	MatchIDs                []int64   `json:"matchIds"`
	From                    time.Time `json:"from"`
	To                      time.Time `json:"to"`
	FetchedAt               time.Time `json:"fetchedAt"`
}

type PositionSnapshot struct {
	Stats  []PositionHeroStat
	Sample PositionSample
}

type positionPlayer struct {
	HeroID      int  `json:"hero_id"`
	PlayerSlot  *int `json:"player_slot"`
	PositionEst *int `json:"position_est"`
	Kills       *int `json:"kills"`
	Deaths      *int `json:"deaths"`
	Assists     *int `json:"assists"`
}

type positionMatch struct {
	MatchID    int64            `json:"match_id"`
	LeagueID   int              `json:"leagueid"`
	StartTime  int64            `json:"start_time"`
	RadiantWin *bool            `json:"radiant_win"`
	Players    []positionPlayer `json:"players"`
}

func (c *Client) HeroPositionStats(ctx context.Context) (PositionSnapshot, error) {
	c.positionsMu.Lock()
	defer c.positionsMu.Unlock()
	if err := ctx.Err(); err != nil {
		return PositionSnapshot{}, err
	}
	if c.positionsCache != nil && time.Now().Before(c.positionsExpiresAt) {
		return clonePositionSnapshot(*c.positionsCache), nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	snapshot, err := c.fetchPositionSnapshot(ctx)
	if err != nil {
		return PositionSnapshot{}, err
	}
	c.positionsCache = &snapshot
	c.positionsExpiresAt = time.Now().Add(c.cacheTTL)
	return clonePositionSnapshot(snapshot), nil
}

func (c *Client) fetchPositionSnapshot(ctx context.Context) (PositionSnapshot, error) {
	var recent []struct {
		MatchID int64 `json:"match_id"`
	}
	if err := c.getPositionJSON(ctx, "/proMatches", &recent); err != nil {
		return PositionSnapshot{}, err
	}
	ids := make([]int64, 0, positionMatchLimit)
	seen := make(map[int64]bool)
	for _, match := range recent {
		if match.MatchID <= 0 || seen[match.MatchID] {
			continue
		}
		seen[match.MatchID] = true
		ids = append(ids, match.MatchID)
		if len(ids) == positionMatchLimit {
			break
		}
	}
	if len(ids) == 0 {
		return PositionSnapshot{}, fmt.Errorf("nenhuma partida profissional disponível para a amostra")
	}

	type matchResult struct {
		match positionMatch
		err   error
	}
	results := make([]matchResult, len(ids))
	jobs := make(chan int, len(ids))
	for i := range ids {
		jobs <- i
	}
	close(jobs)
	var workers sync.WaitGroup
	for range min(4, len(ids)) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for i := range jobs {
				results[i].err = c.getPositionJSON(ctx, "/matches/"+strconv.FormatInt(ids[i], 10), &results[i].match)
			}
		}()
	}
	workers.Wait()
	if err := ctx.Err(); err != nil {
		return PositionSnapshot{}, err
	}

	snapshot := PositionSnapshot{Sample: PositionSample{
		MatchLimit: positionMatchLimit, MatchesRequested: len(ids), MatchIDs: []int64{},
	}}
	counts := make(map[[2]int]PositionHeroStat)
	for i, result := range results {
		match := result.match
		if result.err != nil || match.MatchID != ids[i] || match.LeagueID <= 0 || match.StartTime <= 0 || match.RadiantWin == nil {
			snapshot.Sample.MatchesUnavailable++
			continue
		}
		if !hasCompletePositions(match.Players) {
			snapshot.Sample.MatchesWithoutPositions++
			continue
		}
		snapshot.Sample.MatchesIncluded++
		snapshot.Sample.MatchIDs = append(snapshot.Sample.MatchIDs, match.MatchID)
		start := time.Unix(match.StartTime, 0).UTC()
		if snapshot.Sample.From.IsZero() || start.Before(snapshot.Sample.From) {
			snapshot.Sample.From = start
		}
		if snapshot.Sample.To.IsZero() || start.After(snapshot.Sample.To) {
			snapshot.Sample.To = start
		}
		for _, player := range match.Players {
			key := [2]int{player.HeroID, *player.PositionEst}
			stat := counts[key]
			stat.HeroID, stat.Position = player.HeroID, *player.PositionEst
			stat.Picks++
			if (*player.PlayerSlot < 128) == *match.RadiantWin {
				stat.Wins++
			}
			if player.Kills != nil && player.Deaths != nil && player.Assists != nil {
				stat.Kills += *player.Kills
				stat.Deaths += *player.Deaths
				stat.Assists += *player.Assists
				stat.KDAPicks++
			}
			counts[key] = stat
		}
	}
	if snapshot.Sample.MatchesIncluded == 0 {
		return PositionSnapshot{}, fmt.Errorf("nenhuma partida da amostra possui as cinco posições estimadas em ambos os times")
	}
	heroes, err := c.Heroes(ctx)
	if err != nil {
		return PositionSnapshot{}, fmt.Errorf("consultar catálogo para a amostra por posição: %w", err)
	}
	names := make(map[int]string, len(heroes))
	for _, hero := range heroes {
		names[hero.ID] = hero.LocalizedName
	}
	snapshot.Stats = make([]PositionHeroStat, 0, len(counts))
	for _, stat := range counts {
		stat.LocalizedName = names[stat.HeroID]
		if stat.LocalizedName == "" {
			return PositionSnapshot{}, fmt.Errorf("herói %d da amostra ausente no catálogo", stat.HeroID)
		}
		snapshot.Stats = append(snapshot.Stats, stat)
	}
	slices.SortFunc(snapshot.Stats, func(a, b PositionHeroStat) int {
		if order := cmp.Compare(a.Position, b.Position); order != 0 {
			return order
		}
		return cmp.Compare(a.HeroID, b.HeroID)
	})
	snapshot.Sample.FetchedAt = time.Now().UTC()
	return snapshot, nil
}

func hasCompletePositions(players []positionPlayer) bool {
	if len(players) != 10 {
		return false
	}
	var slots [2][5]bool
	var positions [2][6]bool
	heroes := make(map[int]bool, 10)
	for _, player := range players {
		if player.HeroID <= 0 || heroes[player.HeroID] || player.PlayerSlot == nil || player.PositionEst == nil ||
			*player.PositionEst < 1 || *player.PositionEst > 5 {
			return false
		}
		heroes[player.HeroID] = true
		slot, team := *player.PlayerSlot, 0
		if slot >= 128 {
			slot, team = slot-128, 1
		}
		if slot < 0 || slot > 4 || slots[team][slot] || positions[team][*player.PositionEst] {
			return false
		}
		slots[team][slot], positions[team][*player.PositionEst] = true, true
	}
	return true
}

func clonePositionSnapshot(snapshot PositionSnapshot) PositionSnapshot {
	snapshot.Stats = slices.Clone(snapshot.Stats)
	snapshot.Sample.MatchIDs = slices.Clone(snapshot.Sample.MatchIDs)
	return snapshot
}

func (c *Client) getPositionJSON(ctx context.Context, path string, output any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("consultar OpenDota %s: %w", path, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("OpenDota %s retornou HTTP %d", path, response.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 16<<20)).Decode(output); err != nil {
		return fmt.Errorf("interpretar OpenDota %s: %w", path, err)
	}
	return nil
}
