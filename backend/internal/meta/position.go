package meta

import (
	"context"
	"fmt"

	"github.com/alvsoaress/ancient-analytics-backend/internal/opendota"
)

type Position int

const (
	Carry Position = iota + 1
	Mid
	Offlane
	Support4
	Support5
)

func (p Position) String() string {
	names := [...]string{"", "carry", "mid", "offlane", "support4", "support5"}
	if p < Carry || p > Support5 {
		return ""
	}
	return names[p]
}

func ParsePosition(value string) (Position, error) {
	for position := Carry; position <= Support5; position++ {
		if position.String() == value {
			return position, nil
		}
	}
	return 0, fmt.Errorf("posição inválida: %q", value)
}

type PositionResult struct {
	Heroes []HeroPerformance
	Sample opendota.PositionSample
}

func (s *Service) HeroesByPosition(ctx context.Context, minPicks int, sortBy string, position Position) (PositionResult, error) {
	if minPicks < 0 || position.String() == "" {
		return PositionResult{}, fmt.Errorf("posição ou mínimo de escolhas inválido")
	}
	snapshot, err := s.client.HeroPositionStats(ctx)
	if err != nil {
		return PositionResult{}, fmt.Errorf("consultar amostra profissional por posição: %w", err)
	}
	heroes := make([]HeroPerformance, 0)
	for _, stat := range snapshot.Stats {
		if stat.Position != int(position) {
			continue
		}
		rate, err := WinRate(stat.Wins, stat.Picks)
		if err != nil {
			return PositionResult{}, fmt.Errorf("calcular desempenho do herói %d na posição %s: %w", stat.HeroID, position, err)
		}
		if stat.Picks < minPicks {
			continue
		}
		heroes = append(heroes, HeroPerformance{
			HeroID: stat.HeroID, Name: stat.LocalizedName, Picks: stat.Picks, Wins: stat.Wins, WinRate: rate,
		})
	}
	if err := sortHeroes(heroes, sortBy); err != nil {
		return PositionResult{}, err
	}
	return PositionResult{Heroes: heroes, Sample: snapshot.Sample}, nil
}
