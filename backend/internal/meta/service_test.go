package meta

import (
	"context"
	"errors"
	"testing"

	"github.com/alvsoaress/ancient-analytics-backend/internal/opendota"
)

type fakeStatsClient struct {
	stats []opendota.HeroStat
	err   error
}

func (f fakeStatsClient) HeroStats(
	ctx context.Context,
) ([]opendota.HeroStat, error) {
	return f.stats, f.err
}

func TestServiceHeroes(t *testing.T) {
	client := fakeStatsClient{
		stats: []opendota.HeroStat{
			{
				ID:            1,
				LocalizedName: "Anti-Mage",
				ProPick:       8,
				ProWin:        3,
			},
		},
	}

	service := NewService(client)

	heroes, err := service.Heroes(context.Background(), 0, "")
	if err != nil {
		t.Fatalf("erro ao obter desempenho: %v", err)
	}

	if len(heroes) != 1 {
		t.Fatalf(
			"quantidade de resultados: %d; esperado: 1",
			len(heroes),
		)
	}

	hero := heroes[0]

	if hero.HeroID != 1 {
		t.Errorf("ID recebido: %d; esperado: 1", hero.HeroID)
	}

	if hero.Name != "Anti-Mage" {
		t.Errorf(
			"nome recebido: %q; esperado: Anti-Mage",
			hero.Name,
		)
	}

	if hero.Picks != 8 {
		t.Errorf("escolhas recebidas: %d; esperado: 8", hero.Picks)
	}

	if hero.Wins != 3 {
		t.Errorf("vitórias recebidas: %d; esperado: 3", hero.Wins)
	}

	if hero.WinRate == nil {
		t.Fatal("esperada taxa de vitória disponível")
	}

	if *hero.WinRate != 37.5 {
		t.Errorf(
			"taxa recebida: %v; esperado: 37.5",
			*hero.WinRate,
		)
	}

	if hero.KDA != nil {
		t.Errorf("o agregado geral não informa K/D/A: %v", *hero.KDA)
	}
}

func TestServiceHeroesEstimatesPickRate(t *testing.T) {
	stats := []opendota.HeroStat{{ID: 1, ProPick: 4}, {ID: 2, ProPick: 16}, {ID: 3}}
	heroes, err := NewService(fakeStatsClient{stats: stats}).Heroes(context.Background(), 1, "")
	if err != nil {
		t.Fatal(err)
	}
	// 20 escolhas no total correspondem a 2 partidas; heróis filtrados também entram no total.
	if len(heroes) != 2 || heroes[0].PickRate == nil || *heroes[0].PickRate != 200 ||
		heroes[1].PickRate == nil || *heroes[1].PickRate != 800 {
		t.Fatalf("taxa de escolha inesperada: %+v", heroes)
	}

	heroes, err = NewService(fakeStatsClient{stats: []opendota.HeroStat{{ID: 1}}}).Heroes(context.Background(), 0, "")
	if err != nil || heroes[0].PickRate != nil {
		t.Fatalf("sem escolhas, a taxa deve ser indisponível: %+v; erro: %v", heroes, err)
	}
}

func TestServiceHeroesWithoutPicks(t *testing.T) {
	client := fakeStatsClient{
		stats: []opendota.HeroStat{
			{
				ID:            1,
				LocalizedName: "Anti-Mage",
				ProPick:       0,
				ProWin:        0,
			},
		},
	}

	service := NewService(client)

	heroes, err := service.Heroes(context.Background(), 0, "")
	if err != nil {
		t.Fatalf("erro ao obter desempenho: %v", err)
	}

	if len(heroes) != 1 {
		t.Fatalf(
			"quantidade de resultados: %d; esperado: 1",
			len(heroes),
		)
	}

	if heroes[0].WinRate != nil {
		t.Fatalf(
			"esperada taxa indisponível; recebido: %v",
			*heroes[0].WinRate,
		)
	}
}

func TestServiceHeroesPropagatesClientError(t *testing.T) {
	expectedErr := errors.New("provedor indisponível")

	client := fakeStatsClient{
		err: expectedErr,
	}

	service := NewService(client)

	heroes, err := service.Heroes(context.Background(), 0, "")

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"esperado erro do cliente; recebido: %v",
			err,
		)
	}

	if heroes != nil {
		t.Error("não deveria retornar resultados quando a consulta falha")
	}
}
func TestServiceHeroesFiltersByMinPicks(t *testing.T) {
	client := fakeStatsClient{
		stats: []opendota.HeroStat{
			{
				ID:            1,
				LocalizedName: "Anti-Mage",
				ProPick:       9,
				ProWin:        3,
			},
			{
				ID:            2,
				LocalizedName: "Axe",
				ProPick:       10,
				ProWin:        4,
			},
		},
	}

	service := NewService(client)

	heroes, err := service.Heroes(context.Background(), 10, "")
	if err != nil {
		t.Fatalf("erro ao consultar desempenho: %v", err)
	}

	if len(heroes) != 1 {
		t.Fatalf(
			"quantidade de resultados: %d; esperado: 1",
			len(heroes),
		)
	}

	hero := heroes[0]

	if hero.HeroID != 2 {
		t.Errorf(
			"herói recebido: %d; esperado: 2",
			hero.HeroID,
		)
	}

	if hero.Picks != 10 {
		t.Errorf(
			"escolhas recebidas: %d; esperado: 10",
			hero.Picks,
		)
	}
}
