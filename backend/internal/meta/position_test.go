package meta

import (
	"context"
	"errors"
	"testing"

	"github.com/alvsoaress/ancient-analytics-backend/internal/opendota"
)

func (f fakeStatsClient) HeroPositionStats(context.Context) (opendota.PositionSnapshot, error) {
	return opendota.PositionSnapshot{}, f.err
}

type positionTestClient struct {
	snapshot opendota.PositionSnapshot
	err      error
}

func (f positionTestClient) HeroStats(context.Context) ([]opendota.HeroStat, error) {
	return nil, errors.New("a consulta por posição não deve usar o agregado geral")
}

func (f positionTestClient) HeroPositionStats(context.Context) (opendota.PositionSnapshot, error) {
	return f.snapshot, f.err
}

func TestParsePosition(t *testing.T) {
	for i, value := range []string{"carry", "mid", "offlane", "support4", "support5"} {
		position, err := ParsePosition(value)
		if err != nil || int(position) != i+1 || position.String() != value {
			t.Errorf("posição %q: %v; erro: %v", value, position, err)
		}
	}
	for _, value := range []string{"", "jungle", "3", "support", "Offlane"} {
		if _, err := ParsePosition(value); err == nil {
			t.Errorf("posição inválida aceita: %q", value)
		}
	}
}

func TestServiceHeroesByPosition(t *testing.T) {
	client := positionTestClient{snapshot: opendota.PositionSnapshot{
		Stats: []opendota.PositionHeroStat{
			{HeroID: 1, LocalizedName: "Hero 1", Position: 1, Picks: 2, Wins: 1, Kills: 7, Deaths: 3, Assists: 5, KDAPicks: 2},
			{HeroID: 1, LocalizedName: "Hero 1", Position: 2, Picks: 1, Wins: 1},
			{HeroID: 3, LocalizedName: "Hero 3", Position: 3, Picks: 1, Wins: 0},
			{HeroID: 2, LocalizedName: "Hero 2", Position: 4, Picks: 1, Wins: 1},
			{HeroID: 2, LocalizedName: "Hero 2", Position: 5, Picks: 1, Wins: 0},
		},
		Sample: opendota.PositionSample{MatchesRequested: 3, MatchesIncluded: 2, MatchesUnavailable: 1, MatchIDs: []int64{101, 102}},
	}}
	service := NewService(client)
	for _, test := range []struct {
		position Position
		heroID   int
		rate     float64
	}{
		{Carry, 1, 50}, {Mid, 1, 100}, {Offlane, 3, 0}, {Support4, 2, 100}, {Support5, 2, 0},
	} {
		t.Run(test.position.String(), func(t *testing.T) {
			result, err := service.HeroesByPosition(context.Background(), 1, "winRate", test.position)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Heroes) != 1 || result.Heroes[0].HeroID != test.heroID ||
				result.Heroes[0].WinRate == nil || *result.Heroes[0].WinRate != test.rate {
				t.Fatalf("desempenho inesperado: %+v", result.Heroes)
			}
			if result.Sample.MatchesIncluded != 2 || result.Sample.MatchesUnavailable != 1 {
				t.Fatalf("a cobertura da amostra deve ser preservada: %+v", result.Sample)
			}
		})
	}
	carry, err := service.HeroesByPosition(context.Background(), 1, "", Carry)
	if err != nil {
		t.Fatal(err)
	}
	if hero := carry.Heroes[0]; hero.PickRate == nil || *hero.PickRate != 100 || hero.KDA == nil || *hero.KDA != 4 {
		t.Fatalf("taxa de escolha ou KDA inesperados: %+v", hero)
	}
	mid, err := service.HeroesByPosition(context.Background(), 1, "", Mid)
	if err != nil {
		t.Fatal(err)
	}
	if hero := mid.Heroes[0]; hero.PickRate == nil || *hero.PickRate != 50 || hero.KDA != nil {
		t.Fatalf("sem K/D/A na fonte, o KDA deve ser indisponível: %+v", hero)
	}
	filtered, err := service.HeroesByPosition(context.Background(), 2, "picks", Support4)
	if err != nil || filtered.Heroes == nil || len(filtered.Heroes) != 0 {
		t.Fatalf("filtro deveria retornar lista vazia: %+v; erro: %v", filtered, err)
	}
}

func TestServiceHeroesByPositionErrors(t *testing.T) {
	expected := errors.New("fonte indisponível")
	service := NewService(positionTestClient{err: expected})
	if _, err := service.HeroesByPosition(context.Background(), 0, "", Carry); !errors.Is(err, expected) {
		t.Fatalf("erro original não preservado: %v", err)
	}
	for _, position := range []Position{0, 6} {
		if _, err := service.HeroesByPosition(context.Background(), 0, "", position); err == nil {
			t.Errorf("posição inválida aceita: %d", position)
		}
	}
	if _, err := service.HeroesByPosition(context.Background(), -1, "", Carry); err == nil {
		t.Fatal("mínimo negativo aceito")
	}
	service = NewService(positionTestClient{snapshot: opendota.PositionSnapshot{Stats: []opendota.PositionHeroStat{
		{HeroID: 1, Position: 1, Picks: 1, Wins: 2},
	}}})
	if _, err := service.HeroesByPosition(context.Background(), 0, "", Carry); err == nil {
		t.Fatal("estatísticas inconsistentes não devem ser usadas")
	}
}
