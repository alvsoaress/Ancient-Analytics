package builds

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"github.com/alvsoaress/ancient-analytics-backend/internal/opendota"
)

func guideItem(key string, item opendota.BuildItem) bool {
	if strings.HasPrefix(key, "recipe_") || item.Cost <= 0 {
		return false
	}
	// Moon Shard é consumível, mas representa uma melhoria permanente do herói.
	if item.Quality == "consumable" {
		return key == "moon_shard"
	}
	if len(item.Components) > 0 || key == "boots" {
		return true
	}
	if item.Quality != "component" && item.Quality != "secret_shop" {
		return true
	}
	for _, ability := range item.Abilities {
		if ability.Type == "active" || ability.Type == "toggle" {
			return true
		}
	}
	return false
}
func itemStage(seconds float64) string {
	if seconds <= earlyGameEndsAtSeconds {
		return "early"
	}
	if seconds < lateGameStartsAtSeconds {
		return "core"
	}
	return "late"
}
func medianTime(times []int) float64 {
	slices.Sort(times)
	middle := len(times) / 2
	if len(times)%2 == 0 {
		return (float64(times[middle-1]) + float64(times[middle])) / 2
	}
	return float64(times[middle])
}
func aggregateAlternatives(matches []preparedMatch, catalog opendota.BuildCatalogData) []ItemAlternative {
	type stat struct {
		item  Item
		times []int
		wins  int
	}
	counts := make(map[int]*stat)
	for _, match := range matches {
		for _, purchase := range match.core {
			item := catalog.Items[purchase.Key]
			s := counts[item.ID]
			if s == nil {
				s = &stat{item: itemMetadata(purchase.Key, item)}
				counts[item.ID] = s
			}
			s.times = append(s.times, purchase.Time)
			if match.match.Won {
				s.wins++
			}
		}
	}
	result := make([]ItemAlternative, 0, len(counts))
	for _, s := range counts {
		result = append(result, ItemAlternative{Item: s.item, MedianPurchaseTimeSeconds: medianTime(s.times), PurchaseRate: 100 * float64(len(s.times)) / float64(len(matches)), Performance: performance(len(s.times), s.wins)})
	}
	slices.SortFunc(result, func(a, b ItemAlternative) int {
		if n := cmp.Compare(b.Matches, a.Matches); n != 0 {
			return n
		}
		if n := cmp.Compare(b.Wins, a.Wins); n != 0 {
			return n
		}
		return cmp.Compare(a.Item.ID, b.Item.ID)
	})
	return result
}
func addMatchDetails(result *Build, matches []preparedMatch, catalog opendota.BuildCatalogData) {
	example := matches[0]
	for _, candidate := range matches[1:] {
		a, b := candidate.match, example.match
		if a.Won && !b.Won || a.Won == b.Won && (a.Duration > b.Duration || a.Duration == b.Duration && a.MatchID > b.MatchID) {
			example = candidate
		}
	}
	result.ExampleMatch = ExampleMatch{MatchID: example.match.MatchID, DurationSeconds: example.match.Duration, Won: example.match.Won}
	for _, purchase := range example.purchases {
		result.PurchaseHistory = append(result.PurchaseHistory, PurchaseRecord{Item: itemMetadata(purchase.Key, catalog.Items[purchase.Key]), TimeSeconds: purchase.Time})
	}
	itemsByID := make(map[int]Item, len(catalog.Items))
	for key, item := range catalog.Items {
		itemsByID[item.ID] = itemMetadata(key, item)
	}
	type finalStat struct {
		slots         []*Item
		matches, wins int
	}
	inventories := make(map[string]*finalStat)
	for _, match := range matches {
		ids := match.match.FinalItems
		if len(ids) != 6 {
			continue
		}
		sorted := slices.Clone(ids)
		slices.Sort(sorted)
		parts := make([]string, 0, 6)
		for _, id := range sorted {
			parts = append(parts, strconv.Itoa(id))
		}
		key := strings.Join(parts, "-")
		s := inventories[key]
		if s == nil {
			s = &finalStat{slots: make([]*Item, 0, 6)}
			for _, id := range ids {
				if id == 0 {
					s.slots = append(s.slots, nil)
					continue
				}
				item, ok := itemsByID[id]
				if !ok {
					item = Item{ID: id, Name: "Item " + strconv.Itoa(id)}
				}
				s.slots = append(s.slots, &item)
			}
			inventories[key] = s
		}
		s.matches++
		if match.match.Won {
			s.wins++
		}
	}
	keys := make([]string, 0, len(inventories))
	for key := range inventories {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		s := inventories[key]
		result.FinalInventories = append(result.FinalInventories, FinalInventory{Slots: s.slots, Performance: performance(s.matches, s.wins)})
	}
	slices.SortStableFunc(result.FinalInventories, func(a, b FinalInventory) int { return cmp.Compare(b.Matches, a.Matches) })
}
