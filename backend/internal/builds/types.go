package builds

import "time"

type Result struct {
	Hero             Hero              `json:"hero"`
	Source           string            `json:"source"`
	Scope            string            `json:"scope"`
	Position         *string           `json:"position"`
	PositionSource   *string           `json:"positionSource"`
	Grouping         Grouping          `json:"grouping"`
	Sample           Sample            `json:"sample"`
	AlternativeItems []ItemAlternative `json:"alternativeItems"`
	Builds           []Build           `json:"builds"`
}
type Hero struct {
	HeroID    int                `json:"heroId"`
	Name      string             `json:"name"`
	ImageURL  *string            `json:"imageUrl"`
	Abilities []Ability          `json:"abilities"`
	Talents   []TalentDefinition `json:"talents"`
}
type Grouping struct {
	Method                  string `json:"method"`
	EarlyGameEndsAtSeconds  int    `json:"earlyGameEndsAtSeconds"`
	LateGameStartsAtSeconds int    `json:"lateGameStartsAtSeconds"`
}
type Sample struct {
	MatchLimit                 int        `json:"matchLimit"`
	MatchesRequested           int        `json:"matchesRequested"`
	MatchesUnavailable         int        `json:"matchesUnavailable"`
	MatchesWithoutPurchaseLog  int        `json:"matchesWithoutPurchaseLog"`
	MatchesOutsidePosition     int        `json:"matchesOutsidePosition"`
	MatchesWithoutPosition     int        `json:"matchesWithoutPosition"`
	MatchesIncluded            int        `json:"matchesIncluded"`
	MatchesWithUnknownItems    int        `json:"matchesWithUnknownItems"`
	MatchesWithoutCoreItems    int        `json:"matchesWithoutCoreItems"`
	MatchesGrouped             int        `json:"matchesGrouped"`
	MatchesWithAbilityUpgrades int        `json:"matchesWithAbilityUpgrades"`
	MatchIDs                   []int64    `json:"matchIds"`
	PatchIDs                   []int      `json:"patchIds"`
	From                       *time.Time `json:"from"`
	To                         *time.Time `json:"to"`
	FetchedAt                  time.Time  `json:"fetchedAt"`
}
type Item struct {
	ID       int     `json:"id"`
	Key      string  `json:"key"`
	Name     string  `json:"name"`
	ImageURL *string `json:"imageUrl"`
}
type Ability struct {
	ID       int     `json:"id"`
	Key      *string `json:"key"`
	Name     *string `json:"name"`
	ImageURL *string `json:"imageUrl"`
}
type TalentDefinition struct {
	Level   int     `json:"level"`
	Ability Ability `json:"ability"`
}
type Performance struct {
	Matches int     `json:"matches"`
	Wins    int     `json:"wins"`
	WinRate float64 `json:"winRate"`
}
type Build struct {
	ID string `json:"id"`
	Performance
	PickRate            float64             `json:"pickRate"`
	ExampleMatch        ExampleMatch        `json:"exampleMatch"`
	PurchaseHistory     []PurchaseRecord    `json:"purchaseHistory"`
	FinalInventories    []FinalInventory    `json:"finalInventories"`
	CoreItems           []CoreItem          `json:"coreItems"`
	StartingInventories []StartingInventory `json:"startingInventories"`
	AbilityBuilds       []AbilityBuild      `json:"abilityBuilds"`
	Talents             []TalentChoice      `json:"talents"`
}
type CoreItem struct {
	Stage                     string  `json:"stage"`
	Item                      Item    `json:"item"`
	MedianPurchaseTimeSeconds float64 `json:"medianPurchaseTimeSeconds"`
}
type InventoryItem struct {
	Item     Item `json:"item"`
	Quantity int  `json:"quantity"`
}
type StartingInventory struct {
	Items []InventoryItem `json:"items"`
	Performance
}
type AbilityUpgrade struct {
	Order   int     `json:"order"`
	Ability Ability `json:"ability"`
}
type AbilityBuild struct {
	Upgrades []AbilityUpgrade `json:"upgrades"`
	Performance
}
type TalentChoice struct {
	TalentDefinition
	Performance
}

type ExampleMatch struct {
	MatchID         int64 `json:"matchId"`
	DurationSeconds int   `json:"durationSeconds"`
	Won             bool  `json:"won"`
}
type PurchaseRecord struct {
	Item        Item `json:"item"`
	TimeSeconds int  `json:"timeSeconds"`
}
type FinalInventory struct {
	Slots []*Item `json:"slots"`
	Performance
}
type ItemAlternative struct {
	Item                      Item    `json:"item"`
	MedianPurchaseTimeSeconds float64 `json:"medianPurchaseTimeSeconds"`
	PurchaseRate              float64 `json:"purchaseRate"`
	Performance
}
