package opendota

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

type BuildItemAbility struct {
	Type string `json:"type"`
}

type BuildItem struct {
	Quality    string             `json:"qual"`
	Components []string           `json:"components"`
	Abilities  []BuildItemAbility `json:"abilities"`
	ID         int                `json:"id"`
	Name       string             `json:"dname"`
	Cost       int                `json:"cost"`
	Image      string             `json:"img"`
}
type BuildAbility struct {
	Name  string `json:"dname"`
	Image string `json:"img"`
}
type BuildTalent struct {
	Name string `json:"name"`
	Tier int    `json:"level"`
}
type BuildHeroAbilities struct {
	Abilities []string      `json:"abilities"`
	Talents   []BuildTalent `json:"talents"`
}

// Alguns heróis agrupam nomes alternativos de uma habilidade no mesmo slot.
func (h *BuildHeroAbilities) UnmarshalJSON(data []byte) error {
	var raw struct {
		Abilities []json.RawMessage `json:"abilities"`
		Talents   []BuildTalent     `json:"talents"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	h.Abilities = []string{}
	h.Talents = raw.Talents
	for _, entry := range raw.Abilities {
		var name string
		if err := json.Unmarshal(entry, &name); err == nil && name != "" {
			h.Abilities = append(h.Abilities, name)
			continue
		}
		var names []string
		if err := json.Unmarshal(entry, &names); err != nil || len(names) == 0 {
			return fmt.Errorf("nomes de habilidade inválidos")
		}
		for _, name := range names {
			if name == "" {
				return fmt.Errorf("nome de habilidade vazio")
			}
		}
		h.Abilities = append(h.Abilities, names...)
	}
	return nil
}

type BuildCatalogData struct {
	Items         map[string]BuildItem
	Abilities     map[string]BuildAbility
	AbilityIDs    map[int]string
	HeroAbilities map[string]BuildHeroAbilities
}

func (c *Client) BuildCatalog(ctx context.Context) (BuildCatalogData, error) {
	c.buildCatalogMu.Lock()
	defer c.buildCatalogMu.Unlock()
	if err := ctx.Err(); err != nil {
		return BuildCatalogData{}, err
	}
	if c.buildCatalogCache != nil && time.Now().Before(c.buildCatalogExpiresAt) {
		return cloneBuildCatalog(*c.buildCatalogCache), nil
	}
	var catalog BuildCatalogData
	var rawAbilityIDs map[string]string
	for _, resource := range []struct {
		path   string
		output any
	}{
		{"items", &catalog.Items}, {"abilities", &catalog.Abilities}, {"ability_ids", &rawAbilityIDs}, {"hero_abilities", &catalog.HeroAbilities},
	} {
		if err := c.getBuildJSON(ctx, "/constants/"+resource.path, resource.output); err != nil {
			return BuildCatalogData{}, err
		}
		// Cada recurso é um catálogo, nunca null nem um objeto vazio.
		switch resource.path {
		case "items":
			if len(catalog.Items) == 0 {
				return BuildCatalogData{}, fmt.Errorf("catálogo de itens vazio")
			}
		case "abilities":
			if len(catalog.Abilities) == 0 {
				return BuildCatalogData{}, fmt.Errorf("catálogo de habilidades vazio")
			}
		case "ability_ids":
			if len(rawAbilityIDs) == 0 {
				return BuildCatalogData{}, fmt.Errorf("identificadores de habilidades vazios")
			}
		case "hero_abilities":
			if len(catalog.HeroAbilities) == 0 {
				return BuildCatalogData{}, fmt.Errorf("habilidades dos heróis vazias")
			}
		}
	}
	ids := make(map[int]bool, len(catalog.Items))
	for key, item := range catalog.Items {
		if key == "" || item.ID <= 0 || item.Cost < 0 || ids[item.ID] {
			return BuildCatalogData{}, fmt.Errorf("item inválido no catálogo")
		}
		ids[item.ID] = true
	}
	catalog.AbilityIDs = make(map[int]string, len(rawAbilityIDs))
	// A fonte também fornece chaves com mais de um ID, como "3060,1617".
	for raw, key := range rawAbilityIDs {
		for _, part := range strings.Split(raw, ",") {
			id, err := strconv.ParseUint(part, 10, 31)
			if err != nil || key == "" {
				return BuildCatalogData{}, fmt.Errorf("identificador de habilidade inválido")
			}
			if previous, exists := catalog.AbilityIDs[int(id)]; exists && previous != key {
				return BuildCatalogData{}, fmt.Errorf("identificador de habilidade ambíguo")
			}
			catalog.AbilityIDs[int(id)] = key
		}
	}

	for _, hero := range catalog.HeroAbilities {
		for _, talent := range hero.Talents {
			if talent.Tier < 1 || talent.Name == "" {
				return BuildCatalogData{}, fmt.Errorf("talento inválido no catálogo")
			}
		}
	}
	c.buildCatalogCache = &catalog
	c.buildCatalogExpiresAt = time.Now().Add(c.cacheTTL)
	return cloneBuildCatalog(catalog), nil
}
func cloneBuildCatalog(c BuildCatalogData) BuildCatalogData {
	c.Items = maps.Clone(c.Items)
	for key, item := range c.Items {
		item.Components = slices.Clone(item.Components)
		item.Abilities = slices.Clone(item.Abilities)
		c.Items[key] = item
	}
	c.Abilities = maps.Clone(c.Abilities)
	c.AbilityIDs = maps.Clone(c.AbilityIDs)
	c.HeroAbilities = maps.Clone(c.HeroAbilities)
	for key, h := range c.HeroAbilities {
		h.Abilities = slices.Clone(h.Abilities)
		h.Talents = slices.Clone(h.Talents)
		c.HeroAbilities[key] = h
	}
	return c
}
func (c *Client) getBuildJSON(ctx context.Context, path string, output any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "AncientAnalytics/1.0")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("consultar OpenDota %s: %w", path, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("OpenDota %s retornou HTTP %d", path, response.StatusCode)
	}
	const maxBytes = 16 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return fmt.Errorf("ler OpenDota %s: %w", path, err)
	}
	if len(body) > maxBytes {
		return fmt.Errorf("resposta OpenDota %s excede o limite permitido", path)
	}
	if err := json.Unmarshal(body, output); err != nil {
		return fmt.Errorf("interpretar OpenDota %s: %w", path, err)
	}
	return nil
}
