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
	HeroID   int      `json:"heroId"`
	Name     string   `json:"name"`
	Picks    int      `json:"picks"`
	Wins     int      `json:"wins"`
	WinRate  *float64 `json:"winRate"`
	PickRate *float64 `json:"pickRate"`
	KDA      *float64 `json:"kda"`
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

	// A OpenDota não informa o total de partidas profissionais; cada partida tem dez escolhas,
	// então o total é estimado pela soma das escolhas de todos os heróis dividida por dez.
	totalPicks := 0
	for _, stat := range stats {
		totalPicks += stat.ProPick
	}
	matches := float64(totalPicks) / 10

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
			HeroID:   stat.ID,
			Name:     stat.LocalizedName,
			Picks:    stat.ProPick,
			Wins:     stat.ProWin,
			WinRate:  rate,
			PickRate: PickRate(stat.ProPick, matches),
		})
	}

	if err := sortHeroes(heroes, sortBy); err != nil {
		return nil, err
	}

	return heroes, nil
}
