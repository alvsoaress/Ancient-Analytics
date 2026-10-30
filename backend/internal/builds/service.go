package builds

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/alvsoaress/ancient-analytics-backend/internal/meta"
	"github.com/alvsoaress/ancient-analytics-backend/internal/opendota"
)

const earlyGameEndsAtSeconds = 600
const lateGameStartsAtSeconds = 1500

var (
	ErrInvalidHeroID   = errors.New("heroId inválido")
	ErrInvalidPosition = errors.New("posição inválida")
	ErrNotFound        = errors.New("herói não disponível no catálogo")
	assetPath          = regexp.MustCompile(`^/apps/dota2/images/dota_react/(heroes|items|abilities)/[a-z0-9_]+\.png$`)
)

type Client interface {
	Heroes(context.Context) ([]opendota.Hero, error)
	BuildCatalog(context.Context) (opendota.BuildCatalogData, error)
	HeroBuilds(context.Context, int) (opendota.HeroBuildSnapshot, error)
}
type Service struct{ client Client }

func NewService(client Client) *Service { return &Service{client: client} }

type preparedMatch struct {
	match     opendota.HeroBuildMatch
	core      []opendota.BuildPurchase
	starting  []InventoryItem
	purchases []opendota.BuildPurchase
}

func (s *Service) Hero(ctx context.Context, heroID int, position meta.Position) (Result, error) {
	if heroID <= 0 || heroID > 2147483647 {
		return Result{}, ErrInvalidHeroID
	}
	if position != 0 && position.String() == "" {
		return Result{}, ErrInvalidPosition
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	heroes, err := s.client.Heroes(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("consultar catálogo de heróis: %w", err)
	}
	index := slices.IndexFunc(heroes, func(h opendota.Hero) bool { return h.ID == heroID })
	if index < 0 {
		return Result{}, ErrNotFound
	}
	hero := heroes[index]
	catalog, err := s.client.BuildCatalog(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("consultar metadados de builds: %w", err)
	}
	snapshot, err := s.client.HeroBuilds(ctx, heroID)
	if err != nil {
		return Result{}, fmt.Errorf("consultar amostra de builds: %w", err)
	}
	abilityIDs := make(map[string]int, len(catalog.AbilityIDs))
	for id, key := range catalog.AbilityIDs {
		if previous, ok := abilityIDs[key]; !ok || id < previous {
			abilityIDs[key] = id
		}
	}
	result := Result{
		Hero: heroMetadata(hero, catalog, abilityIDs), Source: "opendota", Scope: "professional_sample",
		Grouping: Grouping{Method: "complete_item_progression", EarlyGameEndsAtSeconds: earlyGameEndsAtSeconds, LateGameStartsAtSeconds: lateGameStartsAtSeconds},
		Sample:   Sample{MatchLimit: snapshot.MatchLimit, MatchesRequested: snapshot.MatchesRequested, MatchesUnavailable: snapshot.MatchesUnavailable, MatchesWithoutPurchaseLog: snapshot.MatchesWithoutPurchaseLog, MatchIDs: []int64{}, PatchIDs: []int{}, FetchedAt: snapshot.FetchedAt}, Builds: []Build{}, AlternativeItems: []ItemAlternative{},
	}
	if position != 0 {
		value, source := position.String(), "opendota.position_est"
		result.Position = &value
		result.PositionSource = &source
	}
	groups := make(map[string][]preparedMatch)
	allPrepared := []preparedMatch{}
	patches := make(map[int]bool)
	for _, match := range snapshot.Matches {
		if position != 0 {
			if match.Position == 0 {
				result.Sample.MatchesWithoutPosition++
				continue
			}
			if match.Position != int(position) {
				result.Sample.MatchesOutsidePosition++
				continue
			}
		}
		sample := &result.Sample
		sample.MatchesIncluded++
		sample.MatchIDs = append(sample.MatchIDs, match.MatchID)
		if len(match.AbilityUpgrades) > 0 {
			sample.MatchesWithAbilityUpgrades++
		}
		if match.Patch != nil {
			patches[*match.Patch] = true
		}
		start := time.Unix(match.StartTime, 0).UTC()
		if sample.From == nil || start.Before(*sample.From) {
			sample.From = &start
		}
		if sample.To == nil || start.After(*sample.To) {
			sample.To = &start
		}
		prepared, known := prepareMatch(match, catalog.Items)
		if !known {
			sample.MatchesWithUnknownItems++
			continue
		}
		if len(prepared.core) == 0 {
			sample.MatchesWithoutCoreItems++
			continue
		}
		ids := make([]string, 0, len(prepared.core))
		for _, purchase := range prepared.core {
			ids = append(ids, strconv.Itoa(catalog.Items[purchase.Key].ID))
		}
		key := strings.Join(ids, "-")
		groups[key] = append(groups[key], prepared)
		allPrepared = append(allPrepared, prepared)
		sample.MatchesGrouped++
	}
	for patch := range patches {
		result.Sample.PatchIDs = append(result.Sample.PatchIDs, patch)
	}
	slices.Sort(result.Sample.PatchIDs)
	for key, matches := range groups {
		result.Builds = append(result.Builds, aggregateBuild(key, matches, catalog, result.Hero.Talents, result.Sample.MatchesGrouped))
	}
	slices.SortFunc(result.Builds, func(a, b Build) int {
		if order := cmp.Compare(b.Matches, a.Matches); order != 0 {
			return order
		}
		if order := cmp.Compare(b.Wins, a.Wins); order != 0 {
			return order
		}
		if order := cmp.Compare(len(b.CoreItems), len(a.CoreItems)); order != 0 {
			return order
		}
		return strings.Compare(a.ID, b.ID)
	})
	result.AlternativeItems = aggregateAlternatives(allPrepared, catalog)
	return result, nil
}
func prepareMatch(match opendota.HeroBuildMatch, items map[string]opendota.BuildItem) (preparedMatch, bool) {
	purchases := slices.Clone(match.Purchases)
	slices.SortStableFunc(purchases, func(a, b opendota.BuildPurchase) int { return cmp.Compare(a.Time, b.Time) })
	result := preparedMatch{match: match, core: []opendota.BuildPurchase{}, starting: []InventoryItem{}, purchases: purchases}
	seen := make(map[int]bool)
	starting := make(map[string]int)
	for _, purchase := range purchases {
		item, ok := items[purchase.Key]
		if !ok {
			return preparedMatch{}, false
		}
		if purchase.Time <= 0 {
			starting[purchase.Key]++
		}
		if purchase.Time <= 0 || !guideItem(purchase.Key, item) || seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		result.core = append(result.core, purchase)
	}
	for key, quantity := range starting {
		result.starting = append(result.starting, InventoryItem{Item: itemMetadata(key, items[key]), Quantity: quantity})
	}
	slices.SortFunc(result.starting, func(a, b InventoryItem) int { return cmp.Compare(a.Item.ID, b.Item.ID) })
	return result, true
}
func itemMetadata(key string, item opendota.BuildItem) Item {
	name := item.Name
	if name == "" {
		name = key
	}
	return Item{ID: item.ID, Key: key, Name: name, ImageURL: steamImage(item.Image, "items")}
}
func abilityMetadata(id int, catalog opendota.BuildCatalogData) Ability {
	result := Ability{ID: id}
	key, ok := catalog.AbilityIDs[id]
	if !ok {
		return result
	}
	result.Key = &key
	definition, ok := catalog.Abilities[key]
	if !ok {
		return result
	}
	if definition.Name != "" {
		name := definition.Name
		result.Name = &name
	}
	result.ImageURL = steamImage(definition.Image, "abilities")
	return result
}
func heroMetadata(hero opendota.Hero, catalog opendota.BuildCatalogData, ids map[string]int) Hero {
	result := Hero{HeroID: hero.ID, Name: hero.LocalizedName, Abilities: []Ability{}, Talents: []TalentDefinition{}}
	if strings.HasPrefix(hero.Name, "npc_dota_hero_") {
		result.ImageURL = steamImage("/apps/dota2/images/dota_react/heroes/"+strings.TrimPrefix(hero.Name, "npc_dota_hero_")+".png", "heroes")
	}
	data := catalog.HeroAbilities[hero.Name]
	for _, key := range data.Abilities {
		if id, ok := ids[key]; ok {
			result.Abilities = append(result.Abilities, abilityMetadata(id, catalog))
		}
	}
	for _, talent := range data.Talents {
		if talent.Tier < 1 || talent.Tier > 4 {
			continue
		}
		if id, ok := ids[talent.Name]; ok {
			result.Talents = append(result.Talents, TalentDefinition{Level: 5 + 5*talent.Tier, Ability: abilityMetadata(id, catalog)})
		}
	}
	return result
}
func steamImage(path, kind string) *string {
	parsed, err := url.Parse(path)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.User != nil || !assetPath.MatchString(parsed.Path) || !strings.HasPrefix(parsed.Path, "/apps/dota2/images/dota_react/"+kind+"/") {
		return nil
	}
	value := "https://cdn.steamstatic.com" + parsed.Path
	return &value
}
func performance(matches, wins int) Performance {
	return Performance{Matches: matches, Wins: wins, WinRate: 100 * float64(wins) / float64(matches)}
}
