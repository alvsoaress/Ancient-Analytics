package meta

import (
	"context"
	"fmt"

	"github.com/alvsoaress/ancient-analytics-backend/internal/opendota"
)

type StatsClient interface {
	HeroStats(ctx context.Context) ([]opendota.HeroStat, error)
	HeroPositionStats(ctx context.Context) (opendota.PositionSnapshot, error)
}

type HeroPerformance struct {
	HeroID  int      `json:"heroId"`
	Name    string   `json:"name"`
	Picks   int      `json:"picks"`
	Wins    int      `json:"wins"`
	WinRate *float64 `json:"winRate"`
}

type Service struct {
	client StatsClient
}

func NewService(client StatsClient) *Service {
	return &Service{
		client: client,
	}
}

func (s *Service) Heroes(
	ctx context.Context,
	minPicks int,
	sortBy string,
) ([]HeroPerformance, error) {
	if minPicks < 0 {
		return nil, fmt.Errorf("minPicks não pode ser negativo")
	}

	stats, err := s.client.HeroStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("consultar estatísticas de meta: %w", err)
	}

	heroes := make([]HeroPerformance, 0, len(stats))

	for _, stat := range stats {
		rate, err := WinRate(stat.ProWin, stat.ProPick)
		if err != nil {
			return nil, fmt.Errorf(
				"calcular desempenho do herói %d: %w",
				stat.ID,
				err,
			)
		}

		if stat.ProPick < minPicks {
			continue
		}

		heroes = append(heroes, HeroPerformance{
			HeroID:  stat.ID,
			Name:    stat.LocalizedName,
			Picks:   stat.ProPick,
			Wins:    stat.ProWin,
			WinRate: rate,
		})
	}

	if err := sortHeroes(heroes, sortBy); err != nil {
		return nil, err
	}

	return heroes, nil
}
