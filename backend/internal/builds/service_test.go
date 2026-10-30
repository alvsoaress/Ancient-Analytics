package builds

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/alvsoaress/ancient-analytics-backend/internal/meta"
	"github.com/alvsoaress/ancient-analytics-backend/internal/opendota"
)

type fakeClient struct {
	heroes   []opendota.Hero
	catalog  opendota.BuildCatalogData
	snapshot opendota.HeroBuildSnapshot
	err      error
	calls    int
}

func (f *fakeClient) Heroes(context.Context) ([]opendota.Hero, error) {
	f.calls++
	return f.heroes, f.err
}
func (f *fakeClient) BuildCatalog(context.Context) (opendota.BuildCatalogData, error) {
	f.calls++
	return f.catalog, f.err
}
func (f *fakeClient) HeroBuilds(context.Context, int) (opendota.HeroBuildSnapshot, error) {
	f.calls++
	return f.snapshot, f.err
}
func fixtureClient() *fakeClient {
	return &fakeClient{heroes: []opendota.Hero{{ID: 62, Name: "npc_dota_hero_bounty_hunter", LocalizedName: "Bounty Hunter"}}, catalog: opendota.BuildCatalogData{
		Items: map[string]opendota.BuildItem{
			"tango":              {ID: 44, Name: "Tango", Cost: 90, Image: "/apps/dota2/images/dota_react/items/tango.png"},
			"blink":              {ID: 1, Name: "Blink Dagger", Cost: 2250, Image: "/apps/dota2/images/dota_react/items/blink.png?t=1"},
			"force_staff":        {ID: 102, Name: "Force Staff", Cost: 2200, Image: "/apps/dota2/images/dota_react/items/force_staff.png"},
			"lotus_orb":          {ID: 226, Name: "Lotus Orb", Cost: 3850},
			"recipe_force_staff": {ID: 101, Name: "Recipe", Cost: 1000},
		},
		Abilities:     map[string]opendota.BuildAbility{"bounty_hunter_track": {Name: "Track", Image: "/apps/dota2/images/dota_react/abilities/bounty_hunter_track.png"}, "special_bonus_attack_damage_30": {Name: "+30 Damage"}},
		AbilityIDs:    map[int]string{5118: "bounty_hunter_track", 6001: "special_bonus_attack_damage_30"},
		HeroAbilities: map[string]opendota.BuildHeroAbilities{"npc_dota_hero_bounty_hunter": {Abilities: []string{"bounty_hunter_track"}, Talents: []opendota.BuildTalent{{Name: "special_bonus_attack_damage_30", Tier: 1}}}},
	}, snapshot: opendota.HeroBuildSnapshot{MatchLimit: 20, MatchesRequested: 3, FetchedAt: time.Unix(5000, 0).UTC(), Matches: []opendota.HeroBuildMatch{
		{MatchID: 30, StartTime: 3000, Position: 4, Won: true, Patch: ptr(50), Purchases: []opendota.BuildPurchase{{Key: "tango", Time: -60}, {Key: "tango", Time: -50}, {Key: "blink", Time: 1000}, {Key: "blink", Time: 1100}, {Key: "recipe_force_staff", Time: 1200}, {Key: "force_staff", Time: 1400}, {Key: "lotus_orb", Time: 2000}}, AbilityUpgrades: []int{5118, 6001}},
		{MatchID: 20, StartTime: 2000, Position: 4, Won: false, Patch: ptr(50), Purchases: []opendota.BuildPurchase{{Key: "tango", Time: 0}, {Key: "force_staff", Time: 1600}, {Key: "blink", Time: 800}, {Key: "lotus_orb", Time: 2200}}, AbilityUpgrades: []int{5118, 6001}},
		{MatchID: 10, StartTime: 1000, Position: 5, Won: true, Patch: ptr(49), Purchases: []opendota.BuildPurchase{{Key: "force_staff", Time: 1500}, {Key: "blink", Time: 1000}}},
	}}}
}
func ptr[T any](v T) *T { return &v }
func TestHeroBuildsGroupsObservedSequencesAndKeepsDenominators(t *testing.T) {
	c := fixtureClient()
	r, err := NewService(c).Hero(context.Background(), 62, meta.Support4)
	if err != nil {
		t.Fatal(err)
	}
	if r.Hero.HeroID != 62 || r.Hero.Name != "Bounty Hunter" || r.Hero.ImageURL == nil || !strings.HasSuffix(*r.Hero.ImageURL, "/heroes/bounty_hunter.png") {
		t.Fatalf("hero: %+v", r.Hero)
	}
	if r.Position == nil || *r.Position != "support4" || r.Scope != "professional_sample" || r.Sample.MatchesIncluded != 2 || r.Sample.MatchesOutsidePosition != 1 || r.Sample.MatchesGrouped != 2 {
		t.Fatalf("sample: %+v", r.Sample)
	}
	if r.Sample.From == nil || r.Sample.From.Unix() != 2000 || r.Sample.To.Unix() != 3000 || !reflect.DeepEqual(r.Sample.MatchIDs, []int64{30, 20}) || !reflect.DeepEqual(r.Sample.PatchIDs, []int{50}) {
		t.Fatalf("period: %+v", r.Sample)
	}
	if len(r.Builds) != 1 {
		t.Fatalf("builds: %+v", r.Builds)
	}
	b := r.Builds[0]
	if b.Matches != 2 || b.Wins != 1 || b.WinRate != 50 || b.PickRate != 100 || len(b.CoreItems) != 3 || b.CoreItems[0].Item.ID != 1 || b.CoreItems[0].MedianPurchaseTimeSeconds != 900 || b.CoreItems[1].MedianPurchaseTimeSeconds != 1500 {
		t.Fatalf("build: %+v", b)
	}
	if b.CoreItems[0].Item.ImageURL == nil || *b.CoreItems[0].Item.ImageURL != "https://cdn.steamstatic.com/apps/dota2/images/dota_react/items/blink.png" {
		t.Fatalf("image: %+v", b.CoreItems[0].Item)
	}
	if len(b.StartingInventories) != 2 || b.StartingInventories[0].Items[0].Quantity+b.StartingInventories[1].Items[0].Quantity != 3 {
		t.Fatalf("starting inventories: %+v", b.StartingInventories)
	}
	if len(b.AbilityBuilds) != 1 || b.AbilityBuilds[0].Matches != 2 || b.AbilityBuilds[0].Upgrades[0].Order != 1 || b.AbilityBuilds[0].Upgrades[0].Ability.ID != 5118 {
		t.Fatalf("abilities: %+v", b.AbilityBuilds)
	}
	if len(b.Talents) != 1 || b.Talents[0].Level != 10 || b.Talents[0].Matches != 2 || b.Talents[0].WinRate != 50 {
		t.Fatalf("talents: %+v", b.Talents)
	}
	if r.Grouping.Method != "complete_item_progression" || r.Grouping.EarlyGameEndsAtSeconds != 600 {
		t.Fatalf("grouping: %+v", r.Grouping)
	}
}
func TestHeroBuildsIncompleteSequencesAndUnknownMetadata(t *testing.T) {
	c := fixtureClient()
	c.snapshot.Matches = c.snapshot.Matches[2:]
	c.snapshot.Matches[0].AbilityUpgrades = []int{999999}
	r, err := NewService(c).Hero(context.Background(), 62, 0)
	if err != nil {
		t.Fatal(err)
	}
	if r.Position != nil || len(r.Builds) != 1 || len(r.Builds[0].CoreItems) != 2 || r.Sample.MatchesWithAbilityUpgrades != 1 {
		t.Fatalf("partial: %+v", r)
	}
	a := r.Builds[0].AbilityBuilds[0].Upgrades[0].Ability
	if a.ID != 999999 || a.Name != nil || a.ImageURL != nil {
		t.Fatalf("unknown ability replaced: %+v", a)
	}
	if r.Builds[0].StartingInventories == nil || r.Builds[0].Talents == nil {
		t.Fatal("collections must be arrays")
	}
}
func TestHeroBuildsUnknownItemsAreReportedAndNotGuessed(t *testing.T) {
	c := fixtureClient()
	c.snapshot.Matches[0].Purchases = append(c.snapshot.Matches[0].Purchases, opendota.BuildPurchase{Key: "future_item", Time: 1500})
	r, err := NewService(c).Hero(context.Background(), 62, meta.Support4)
	if err != nil {
		t.Fatal(err)
	}
	if r.Sample.MatchesWithUnknownItems != 1 || r.Sample.MatchesIncluded != 2 || r.Sample.MatchesGrouped != 1 || r.Builds[0].Matches != 1 {
		t.Fatalf("unknown items: %+v", r.Sample)
	}
}
func TestHeroBuildsEmptyDataAndMissingPositions(t *testing.T) {
	c := fixtureClient()
	c.snapshot.Matches[0].Position = 0
	c.snapshot.Matches = c.snapshot.Matches[:1]
	r, err := NewService(c).Hero(context.Background(), 62, meta.Support4)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Builds) != 0 || r.Builds == nil || r.Sample.MatchesWithoutPosition != 1 || r.Sample.MatchesIncluded != 0 || r.Sample.From != nil || r.Sample.To != nil || r.Sample.MatchIDs == nil {
		t.Fatalf("empty: %+v", r)
	}
	c.snapshot.Matches[0].Position = 4
	c.snapshot.Matches[0].Purchases = []opendota.BuildPurchase{{Key: "tango", Time: -1}}
	r, err = NewService(c).Hero(context.Background(), 62, 0)
	if err != nil {
		t.Fatal(err)
	}
	if r.Sample.MatchesWithoutCoreItems != 1 || len(r.Builds) != 0 {
		t.Fatalf("no core items: %+v", r.Sample)
	}
}
func TestHeroBuildsValidatesBeforeConsultingSource(t *testing.T) {
	for _, id := range []int{0, -1, 2147483648} {
		c := fixtureClient()
		_, err := NewService(c).Hero(context.Background(), id, 0)
		if !errors.Is(err, ErrInvalidHeroID) || c.calls != 0 {
			t.Fatalf("id %d: %v calls %d", id, err, c.calls)
		}
	}
	c := fixtureClient()
	_, err := NewService(c).Hero(context.Background(), 62, meta.Position(9))
	if !errors.Is(err, ErrInvalidPosition) || c.calls != 0 {
		t.Fatal("invalid position reached source")
	}
	c = fixtureClient()
	_, err = NewService(c).Hero(context.Background(), 999, 0)
	if !errors.Is(err, ErrNotFound) || c.calls != 1 {
		t.Fatalf("unknown hero: %v calls %d", err, c.calls)
	}
	c = fixtureClient()
	c.err = errors.New("source failure")
	if _, err := NewService(c).Hero(context.Background(), 62, 0); err == nil {
		t.Fatal("source error hidden")
	}
}
func TestHeroBuildsDoesNotMutateCachedSourceAndImagesAreSafe(t *testing.T) {
	c := fixtureClient()
	before := append([]opendota.BuildPurchase(nil), c.snapshot.Matches[1].Purchases...)
	c.catalog.Items["blink"] = opendota.BuildItem{ID: 1, Name: "Blink Dagger", Cost: 2250, Image: "https://evil.example/blink.png"}
	r, err := NewService(c).Hero(context.Background(), 62, meta.Support4)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, c.snapshot.Matches[1].Purchases) || r.Builds[0].CoreItems[0].Item.ImageURL != nil {
		t.Fatal("cache mutated or untrusted image exposed")
	}
}

func TestHeroMetadataDoesNotTreatExtraAbilitySlotsAsLevel30Talents(t *testing.T) {
	c := fixtureClient()
	c.catalog.AbilityIDs[999] = "rubick_hidden3"
	h := c.catalog.HeroAbilities["npc_dota_hero_bounty_hunter"]
	h.Talents = append(h.Talents, opendota.BuildTalent{Name: "rubick_hidden3", Tier: 5})
	c.catalog.HeroAbilities["npc_dota_hero_bounty_hunter"] = h
	r, err := NewService(c).Hero(context.Background(), 62, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Hero.Talents) != 1 {
		t.Fatalf("extra slot became a talent: %+v", r.Hero.Talents)
	}
}

func TestBuildOrderingIsDeterministicAndOnlyCountsGroupedMatches(t *testing.T) {
	c := fixtureClient()
	c.snapshot.Matches = append(c.snapshot.Matches, opendota.HeroBuildMatch{MatchID: 9, StartTime: 900, Won: false, Position: 4, Purchases: []opendota.BuildPurchase{{Key: "tango", Time: -5}}})
	r, err := NewService(c).Hero(context.Background(), 62, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Builds) != 2 || r.Builds[0].ID != "1-102-226" || r.Builds[0].Matches != 2 || r.Sample.MatchesIncluded != 4 || r.Sample.MatchesGrouped != 3 || r.Sample.MatchesWithoutCoreItems != 1 {
		t.Fatalf("ordering/denominator: %+v", r)
	}
	if r.Builds[0].PickRate != 200.0/3 || r.Builds[1].PickRate != 100.0/3 {
		t.Fatalf("pick rates: %+v", r.Builds)
	}
	for range 10 {
		again, err := NewService(c).Hero(context.Background(), 62, 0)
		if err != nil || !reflect.DeepEqual(r, again) {
			t.Fatal("output order changed across requests")
		}
	}
}
func TestSteamImagesRejectExternalAndUnexpectedPaths(t *testing.T) {
	for _, path := range []string{"https://evil.example/image.png", "//evil.example/image.png", "/apps/dota2/images/dota_react/items/../../evil.png", "/apps/dota2/images/dota_react/abilities/test.png", "javascript:alert(1)", "/apps/dota2/images/dota_react/items/test.svg", "/apps/dota2/images/dota_react/items/test.png/extra"} {
		if steamImage(path, "items") != nil {
			t.Errorf("unsafe image accepted: %s", path)
		}
	}
}

func TestBuildProgressionKeepsBootsAndPurchasesAfterTheThirdItem(t *testing.T) {
	c := fixtureClient()
	c.catalog.Items["boots"] = opendota.BuildItem{ID: 29, Name: "Boots of Speed", Cost: 500}
	c.catalog.Items["magic_wand"] = opendota.BuildItem{ID: 36, Name: "Magic Wand", Cost: 450}
	c.catalog.Items["black_king_bar"] = opendota.BuildItem{ID: 116, Name: "Black King Bar", Cost: 4050}
	c.catalog.Items["heart"] = opendota.BuildItem{ID: 114, Name: "Heart of Tarrasque", Cost: 5100}
	c.snapshot.Matches = c.snapshot.Matches[:1]
	c.snapshot.Matches[0].Purchases = []opendota.BuildPurchase{{Key: "tango", Time: -10}, {Key: "boots", Time: 120}, {Key: "magic_wand", Time: 300}, {Key: "blink", Time: 900}, {Key: "force_staff", Time: 1200}, {Key: "black_king_bar", Time: 1500}, {Key: "lotus_orb", Time: 1800}, {Key: "heart", Time: 2400}}
	r, err := NewService(c).Hero(context.Background(), 62, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, entry := range r.Builds[0].CoreItems {
		got = append(got, entry.Item.Key)
	}
	want := []string{"boots", "magic_wand", "blink", "force_staff", "black_king_bar", "lotus_orb", "heart"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("progressão truncada: recebido %v; esperado %v", got, want)
	}
}

func TestBuildGuideExcludesIngredientsAndKeepsLateItemsAndFullSkills(t *testing.T) {
	c := fixtureClient()
	c.catalog.Items["relic"] = opendota.BuildItem{ID: 54, Name: "Sacred Relic", Cost: 3400, Quality: "secret_shop"}
	c.catalog.Items["radiance"] = opendota.BuildItem{ID: 137, Name: "Radiance", Cost: 4700, Quality: "epic", Components: []string{"relic", "talisman_of_evasion"}}
	c.catalog.Items["boots"] = opendota.BuildItem{ID: 29, Name: "Boots of Speed", Cost: 500, Quality: "component"}
	c.catalog.Items["tango"] = opendota.BuildItem{ID: 44, Name: "Tango", Cost: 90, Quality: "consumable"}
	c.snapshot.Matches = c.snapshot.Matches[:1]
	c.snapshot.Matches[0].Purchases = []opendota.BuildPurchase{{Key: "tango", Time: -10}, {Key: "boots", Time: 120}, {Key: "relic", Time: 700}, {Key: "radiance", Time: 900}, {Key: "blink", Time: 1300}, {Key: "lotus_orb", Time: 2100}, {Key: "tango", Time: 2200}}
	c.snapshot.Matches[0].FinalItems = []int{137, 1, 226, 29, 0, 0}
	c.snapshot.Matches[0].Duration = 2500
	c.snapshot.Matches[0].AbilityUpgrades = []int{5118, 5118, 5118, 5118, 5118, 5118, 5118, 5118, 5118, 5118, 6001, 5118, 5118, 5118, 5118}
	r, err := NewService(c).Hero(context.Background(), 62, 0)
	if err != nil {
		t.Fatal(err)
	}
	b := r.Builds[0]
	keys := []string{}
	for _, x := range b.CoreItems {
		keys = append(keys, x.Item.Key)
	}
	if !reflect.DeepEqual(keys, []string{"boots", "radiance", "blink", "lotus_orb"}) {
		t.Fatalf("guide includes components or omits progress: %v", keys)
	}
	if len(b.AbilityBuilds[0].Upgrades) != 15 {
		t.Fatalf("skill order cut at %d", len(b.AbilityBuilds[0].Upgrades))
	}
	if len(b.PurchaseHistory) != 7 || len(b.FinalInventories) != 1 || len(b.FinalInventories[0].Slots) != 6 || b.FinalInventories[0].Slots[0].ID != 137 || b.FinalInventories[0].Slots[4] != nil {
		t.Fatalf("missing full purchases or final inventory: %+v", b)
	}
	if b.CoreItems[0].Stage != "early" || b.CoreItems[1].Stage != "core" || b.CoreItems[3].Stage != "late" {
		t.Fatalf("missing game phases: %+v", b.CoreItems)
	}
}

func TestBuildsKeepDifferentLateProgressionsSeparate(t *testing.T) {
	c := fixtureClient()
	c.snapshot.Matches = c.snapshot.Matches[:2]
	prefix := []opendota.BuildPurchase{{Key: "blink", Time: 700}, {Key: "force_staff", Time: 1000}, {Key: "lotus_orb", Time: 1600}}
	c.catalog.Items["heart"] = opendota.BuildItem{ID: 114, Name: "Heart", Cost: 5100}
	c.catalog.Items["black_king_bar"] = opendota.BuildItem{ID: 116, Name: "BKB", Cost: 4050}
	c.snapshot.Matches[0].Purchases = append(append([]opendota.BuildPurchase{}, prefix...), opendota.BuildPurchase{Key: "heart", Time: 2000})
	c.snapshot.Matches[1].Purchases = append(append([]opendota.BuildPurchase{}, prefix...), opendota.BuildPurchase{Key: "black_king_bar", Time: 2100})
	r, err := NewService(c).Hero(context.Background(), 62, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Builds) != 2 || len(r.Builds[0].CoreItems) != 4 || len(r.Builds[1].CoreItems) != 4 {
		t.Fatalf("different late builds combined or truncated: %+v", r.Builds)
	}
	for _, item := range r.AlternativeItems {
		if item.Item.Key == "blink" && (item.Matches != 2 || item.PurchaseRate != 100) {
			t.Fatalf("incorrect presence: %+v", item)
		}
		if item.Item.Key == "heart" && (item.Matches != 1 || item.PurchaseRate != 50) {
			t.Fatalf("incorrect alternative: %+v", item)
		}
	}
}

func TestGuidePreservesUnknownFinalItemsAndDoesNotInventMissingSlots(t *testing.T) {
	c := fixtureClient()
	c.snapshot.Matches = c.snapshot.Matches[:1]
	r, err := NewService(c).Hero(context.Background(), 62, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Builds[0].FinalInventories) != 0 {
		t.Fatal("missing source inventory became empty slots")
	}
	c.snapshot.Matches[0].FinalItems = []int{999999, 1, 1, 0, 0, 0}
	r, err = NewService(c).Hero(context.Background(), 62, 0)
	if err != nil {
		t.Fatal(err)
	}
	slots := r.Builds[0].FinalInventories[0].Slots
	if len(slots) != 6 || slots[0].ID != 999999 || slots[0].ImageURL != nil || slots[1].ID != 1 || slots[2].ID != 1 || slots[3] != nil {
		t.Fatalf("inventory guessed or duplicates lost: %+v", slots)
	}
}
