package builds

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"github.com/alvsoaress/ancient-analytics-backend/internal/opendota"
)

func aggregateBuild(id string, matches []preparedMatch, catalog opendota.BuildCatalogData, talents []TalentDefinition, total int) Build {
	result := Build{ID: id, PickRate: 100 * float64(len(matches)) / float64(total), PurchaseHistory: []PurchaseRecord{}, FinalInventories: []FinalInventory{}, CoreItems: []CoreItem{}, StartingInventories: []StartingInventory{}, AbilityBuilds: []AbilityBuild{}, Talents: []TalentChoice{}}
	wins := 0
	for _, m := range matches {
		if m.match.Won {
			wins++
		}
	}
	result.Performance = performance(len(matches), wins)
	for i, purchase := range matches[0].core {
		times := make([]int, 0, len(matches))
		for _, match := range matches {
			times = append(times, match.core[i].Time)
		}
		slices.Sort(times)
		middle := len(times) / 2
		median := float64(times[middle])
		if len(times)%2 == 0 {
			median = (float64(times[middle-1]) + float64(times[middle])) / 2
		}
		result.CoreItems = append(result.CoreItems, CoreItem{Stage: itemStage(median), Item: itemMetadata(purchase.Key, catalog.Items[purchase.Key]), MedianPurchaseTimeSeconds: median})
	}
	type inventoryCount struct {
		items         []InventoryItem
		matches, wins int
	}
	inventories := make(map[string]*inventoryCount)
	type abilityCount struct {
		ids           []int
		matches, wins int
	}
	abilities := make(map[string]*abilityCount)
	type talentCount struct {
		definition    TalentDefinition
		matches, wins int
	}
	talentStats := make(map[int]*talentCount)
	definitions := make(map[int]TalentDefinition)
	for _, t := range talents {
		definitions[t.Ability.ID] = t
	}
	for _, match := range matches {
		keyParts := make([]string, 0, len(match.starting))
		for _, item := range match.starting {
			keyParts = append(keyParts, strconv.Itoa(item.Item.ID)+"x"+strconv.Itoa(item.Quantity))
		}
		key := strings.Join(keyParts, "-")
		inventory := inventories[key]
		if inventory == nil {
			inventory = &inventoryCount{items: match.starting}
			inventories[key] = inventory
		}
		inventory.matches++
		if match.match.Won {
			inventory.wins++
		}
		upgrades := match.match.AbilityUpgrades
		if len(upgrades) > 0 {
			parts := make([]string, 0, len(upgrades))
			for _, id := range upgrades {
				parts = append(parts, strconv.Itoa(id))
			}
			key = strings.Join(parts, "-")
			group := abilities[key]
			if group == nil {
				group = &abilityCount{ids: upgrades}
				abilities[key] = group
			}
			group.matches++
			if match.match.Won {
				group.wins++
			}
		}
		seen := make(map[int]bool)
		for _, id := range match.match.AbilityUpgrades {
			definition, ok := definitions[id]
			if !ok || seen[id] {
				continue
			}
			seen[id] = true
			stat := talentStats[id]
			if stat == nil {
				stat = &talentCount{definition: definition}
				talentStats[id] = stat
			}
			stat.matches++
			if match.match.Won {
				stat.wins++
			}
		}
	}
	inventoryKeys := make([]string, 0, len(inventories))
	for key := range inventories {
		inventoryKeys = append(inventoryKeys, key)
	}
	slices.Sort(inventoryKeys)
	for _, key := range inventoryKeys {
		group := inventories[key]
		result.StartingInventories = append(result.StartingInventories, StartingInventory{Items: group.items, Performance: performance(group.matches, group.wins)})
	}
	slices.SortStableFunc(result.StartingInventories, func(a, b StartingInventory) int { return cmp.Compare(b.Matches, a.Matches) })
	abilityKeys := make([]string, 0, len(abilities))
	for key := range abilities {
		abilityKeys = append(abilityKeys, key)
	}
	slices.Sort(abilityKeys)
	for _, key := range abilityKeys {
		group := abilities[key]
		build := AbilityBuild{Upgrades: []AbilityUpgrade{}, Performance: performance(group.matches, group.wins)}
		for i, id := range group.ids {
			build.Upgrades = append(build.Upgrades, AbilityUpgrade{Order: i + 1, Ability: abilityMetadata(id, catalog)})
		}
		result.AbilityBuilds = append(result.AbilityBuilds, build)
	}
	slices.SortStableFunc(result.AbilityBuilds, func(a, b AbilityBuild) int { return cmp.Compare(b.Matches, a.Matches) })
	for _, stat := range talentStats {
		result.Talents = append(result.Talents, TalentChoice{TalentDefinition: stat.definition, Performance: performance(stat.matches, stat.wins)})
	}
	slices.SortFunc(result.Talents, func(a, b TalentChoice) int {
		if order := cmp.Compare(a.Level, b.Level); order != 0 {
			return order
		}
		return cmp.Compare(a.Ability.ID, b.Ability.ID)
	})
	addMatchDetails(&result, matches, catalog)
	return result
}
