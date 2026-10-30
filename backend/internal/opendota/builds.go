package opendota

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"
)

const buildMatchLimit = 20
const buildCacheLimit = 128

type BuildPurchase struct {
	Key  string `json:"key"`
	Time int    `json:"time"`
}
type HeroBuildMatch struct {
	Duration        int
	FinalItems      []int
	MatchID         int64
	StartTime       int64
	Patch           *int
	Position        int
	Won             bool
	Purchases       []BuildPurchase
	AbilityUpgrades []int
}
type HeroBuildSnapshot struct {
	MatchLimit                int
	MatchesRequested          int
	MatchesUnavailable        int
	MatchesWithoutPurchaseLog int
	Matches                   []HeroBuildMatch
	FetchedAt                 time.Time
}
type buildSampleCacheEntry struct {
	data      HeroBuildSnapshot
	expiresAt time.Time
}
type buildMatchData struct {
	MatchID    int64 `json:"match_id"`
	LeagueID   int   `json:"leagueid"`
	StartTime  int64 `json:"start_time"`
	Duration   *int  `json:"duration"`
	Patch      *int  `json:"patch"`
	RadiantWin *bool `json:"radiant_win"`
	Players    []*struct {
		HeroID    int  `json:"hero_id"`
		Slot      *int `json:"player_slot"`
		Position  *int `json:"position_est"`
		Purchases []*struct {
			Key  string `json:"key"`
			Time *int   `json:"time"`
		} `json:"purchase_log"`
		AbilityUpgrades []int `json:"ability_upgrades_arr"`
		Item0           *int  `json:"item_0"`
		Item1           *int  `json:"item_1"`
		Item2           *int  `json:"item_2"`
		Item3           *int  `json:"item_3"`
		Item4           *int  `json:"item_4"`
		Item5           *int  `json:"item_5"`
	} `json:"players"`
}

func (c *Client) HeroBuilds(ctx context.Context, heroID int) (HeroBuildSnapshot, error) {
	if heroID <= 0 || heroID > 2147483647 {
		return HeroBuildSnapshot{}, fmt.Errorf("heroId inválido")
	}
	c.buildSamplesMu.Lock()
	defer c.buildSamplesMu.Unlock()
	if err := ctx.Err(); err != nil {
		return HeroBuildSnapshot{}, err
	}
	if entry, ok := c.buildSamplesCache[heroID]; ok && time.Now().Before(entry.expiresAt) {
		return cloneHeroBuildSnapshot(entry.data), nil
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	snapshot, err := c.fetchHeroBuilds(ctx, heroID)
	if err != nil {
		return HeroBuildSnapshot{}, err
	}
	if c.buildSamplesCache == nil {
		c.buildSamplesCache = make(map[int]buildSampleCacheEntry)
	}
	now := time.Now()
	for id, entry := range c.buildSamplesCache {
		if !now.Before(entry.expiresAt) {
			delete(c.buildSamplesCache, id)
		}
	}
	if len(c.buildSamplesCache) >= buildCacheLimit {
		var oldestID int
		var expiry time.Time
		for id, entry := range c.buildSamplesCache {
			if expiry.IsZero() || entry.expiresAt.Before(expiry) {
				oldestID, expiry = id, entry.expiresAt
			}
		}
		delete(c.buildSamplesCache, oldestID)
	}
	c.buildSamplesCache[heroID] = buildSampleCacheEntry{data: snapshot, expiresAt: now.Add(c.cacheTTL)}
	return cloneHeroBuildSnapshot(snapshot), nil
}
func (c *Client) fetchHeroBuilds(ctx context.Context, heroID int) (HeroBuildSnapshot, error) {
	var recent []struct {
		MatchID int64 `json:"match_id"`
	}
	if err := c.getBuildJSON(ctx, "/heroes/"+strconv.Itoa(heroID)+"/matches", &recent); err != nil {
		return HeroBuildSnapshot{}, err
	}
	if recent == nil {
		return HeroBuildSnapshot{}, fmt.Errorf("lista de partidas inválida")
	}
	ids := make([]int64, 0, buildMatchLimit)
	seen := make(map[int64]bool)
	for _, match := range recent {
		if match.MatchID <= 0 || match.MatchID > 9007199254740991 || seen[match.MatchID] {
			continue
		}
		seen[match.MatchID] = true
		ids = append(ids, match.MatchID)
		if len(ids) == buildMatchLimit {
			break
		}
	}
	if len(recent) > 0 && len(ids) == 0 {
		return HeroBuildSnapshot{}, fmt.Errorf("lista sem identificadores de partidas válidos")
	}
	type matchResult struct {
		data buildMatchData
		err  error
	}
	results := make([]matchResult, len(ids))
	jobs := make(chan int, len(ids))
	for i := range ids {
		jobs <- i
	}
	close(jobs)
	var workers sync.WaitGroup
	for range min(4, len(ids)) {
		workers.Go(func() {
			for i := range jobs {
				results[i].err = c.getBuildJSON(ctx, "/matches/"+strconv.FormatInt(ids[i], 10), &results[i].data)
			}
		})
	}
	workers.Wait()
	if err := ctx.Err(); err != nil {
		return HeroBuildSnapshot{}, err
	}
	snapshot := HeroBuildSnapshot{MatchLimit: buildMatchLimit, MatchesRequested: len(ids), Matches: []HeroBuildMatch{}, FetchedAt: time.Now().UTC()}
	for i, result := range results {
		if result.err != nil {
			snapshot.MatchesUnavailable++
			continue
		}
		match, missingLog, err := normalizeBuildMatch(result.data, ids[i], heroID)
		if err != nil {
			snapshot.MatchesUnavailable++
			continue
		}
		if missingLog {
			snapshot.MatchesWithoutPurchaseLog++
			continue
		}
		snapshot.Matches = append(snapshot.Matches, match)
	}
	if len(ids) > 0 && snapshot.MatchesUnavailable == len(ids) {
		return HeroBuildSnapshot{}, fmt.Errorf("nenhuma partida profissional da amostra pôde ser consultada")
	}
	return snapshot, nil
}
func normalizeBuildMatch(data buildMatchData, id int64, heroID int) (HeroBuildMatch, bool, error) {
	invalid := fmt.Errorf("dados de partida profissional inválidos")
	if data.MatchID != id || data.LeagueID <= 0 || data.StartTime <= 0 || data.StartTime > 253402300799 || data.Duration == nil || *data.Duration < 0 || data.RadiantWin == nil || len(data.Players) > 256 || data.Patch != nil && *data.Patch < 0 {
		return HeroBuildMatch{}, false, invalid
	}
	found := -1
	for i, p := range data.Players {
		if p != nil && p.HeroID == heroID {
			if found != -1 {
				return HeroBuildMatch{}, false, invalid
			}
			found = i
		}
	}
	if found == -1 {
		return HeroBuildMatch{}, false, invalid
	}
	p := data.Players[found]
	if p.Slot == nil || !(*p.Slot >= 0 && *p.Slot <= 4 || *p.Slot >= 128 && *p.Slot <= 132) {
		return HeroBuildMatch{}, false, invalid
	}
	if len(p.Purchases) > 20000 || len(p.AbilityUpgrades) > 128 {
		return HeroBuildMatch{}, false, invalid
	}
	for _, ability := range p.AbilityUpgrades {
		if ability <= 0 {
			return HeroBuildMatch{}, false, invalid
		}
	}
	for _, entry := range p.Purchases {
		if entry == nil || entry.Key == "" || len(entry.Key) > 128 || entry.Time == nil || *entry.Time < -86400 || *entry.Time > *data.Duration+60 {
			return HeroBuildMatch{}, false, invalid
		}
	}
	if p.Purchases == nil {
		return HeroBuildMatch{}, true, nil
	}
	match := HeroBuildMatch{Duration: *data.Duration, MatchID: id, StartTime: data.StartTime, Patch: data.Patch, Won: (*p.Slot < 128) == *data.RadiantWin, AbilityUpgrades: slices.Clone(p.AbilityUpgrades), Purchases: make([]BuildPurchase, 0, len(p.Purchases))}
	slots := []*int{p.Item0, p.Item1, p.Item2, p.Item3, p.Item4, p.Item5}
	complete := true
	for _, slot := range slots {
		if slot == nil {
			complete = false
			continue
		}
		if *slot < 0 {
			return HeroBuildMatch{}, false, invalid
		}
	}
	if complete {
		match.FinalItems = make([]int, 0, 6)
		for _, slot := range slots {
			match.FinalItems = append(match.FinalItems, *slot)
		}
	}
	if p.Position != nil && *p.Position >= 1 && *p.Position <= 5 {
		match.Position = *p.Position
	}
	for _, entry := range p.Purchases {
		match.Purchases = append(match.Purchases, BuildPurchase{Key: entry.Key, Time: *entry.Time})
	}
	return match, false, nil
}
func cloneHeroBuildSnapshot(s HeroBuildSnapshot) HeroBuildSnapshot {
	s.Matches = slices.Clone(s.Matches)
	for i := range s.Matches {
		m := &s.Matches[i]
		m.FinalItems = slices.Clone(m.FinalItems)
		m.Purchases = slices.Clone(m.Purchases)
		m.AbilityUpgrades = slices.Clone(m.AbilityUpgrades)
		if m.Patch != nil {
			patch := *m.Patch
			m.Patch = &patch
		}
	}
	return s
}
