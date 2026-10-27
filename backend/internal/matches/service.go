package matches

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/alvsoaress/ancient-analytics-backend/internal/opendota"
)

// Mantém o identificador exato quando o JSON é consumido pelo JavaScript.
const MaxMatchID int64 = 9007199254740991

var (
	ErrInvalidMatchID = errors.New("matchId inválido")
	ErrNotFound       = errors.New("partida não disponível na OpenDota")
)

type MatchClient interface {
	Match(ctx context.Context, matchID int64) (opendota.MatchData, error)
}

type Service struct{ client MatchClient }

func NewService(client MatchClient) *Service { return &Service{client: client} }

type Details struct {
	MatchID         int64    `json:"matchId"`
	DurationSeconds *int     `json:"durationSeconds"`
	StartTime       *int64   `json:"startTime"`
	GameMode        *int     `json:"gameMode"`
	LobbyType       *int     `json:"lobbyType"`
	Winner          *string  `json:"winner"`
	RadiantScore    *int     `json:"radiantScore"`
	DireScore       *int     `json:"direScore"`
	Players         []Player `json:"players"`
}

type Player struct {
	AccountID     *uint32 `json:"accountId"`
	Name          *string `json:"name"`
	HeroID        *int    `json:"heroId"`
	PlayerSlot    *int    `json:"playerSlot"`
	Team          *string `json:"team"`
	Won           *bool   `json:"won"`
	Kills         *int    `json:"kills"`
	Deaths        *int    `json:"deaths"`
	Assists       *int    `json:"assists"`
	GoldPerMinute *int    `json:"goldPerMinute"`
	XPPerMinute   *int    `json:"xpPerMinute"`
	HeroDamage    *int    `json:"heroDamage"`
	TowerDamage   *int    `json:"towerDamage"`
	HeroHealing   *int    `json:"heroHealing"`
	LastHits      *int    `json:"lastHits"`
	Denies        *int    `json:"denies"`
	Level         *int    `json:"level"`
}

func (s *Service) Details(ctx context.Context, matchID int64) (Details, error) {
	if matchID <= 0 || matchID > MaxMatchID {
		return Details{}, ErrInvalidMatchID
	}
	data, err := s.client.Match(ctx, matchID)
	if errors.Is(err, opendota.ErrMatchNotFound) {
		return Details{}, ErrNotFound
	}
	if err != nil {
		return Details{}, fmt.Errorf("consultar detalhes da partida: %w", err)
	}
	if data.MatchID != matchID {
		return Details{}, fmt.Errorf("partida retornada não corresponde ao matchId solicitado")
	}
	result := Details{
		MatchID:         matchID,
		DurationSeconds: data.Duration,
		StartTime:       data.StartTime,
		GameMode:        data.GameMode,
		LobbyType:       data.LobbyType,
		RadiantScore:    data.RadiantScore,
		DireScore:       data.DireScore,
		Players:         make([]Player, 0, len(data.Players)),
	}
	if data.RadiantWin != nil {
		winner := "dire"
		if *data.RadiantWin {
			winner = "radiant"
		}
		result.Winner = &winner
	}
	for _, p := range data.Players {
		if p == nil {
			return Details{}, fmt.Errorf("jogador da partida indisponível")
		}
		item := Player{
			AccountID:     p.AccountID,
			Name:          p.PersonaName,
			HeroID:        p.HeroID,
			PlayerSlot:    p.PlayerSlot,
			Kills:         p.Kills,
			Deaths:        p.Deaths,
			Assists:       p.Assists,
			GoldPerMinute: p.GoldPerMinute,
			XPPerMinute:   p.XPPerMinute,
			HeroDamage:    p.HeroDamage,
			TowerDamage:   p.TowerDamage,
			HeroHealing:   p.HeroHealing,
			LastHits:      p.LastHits,
			Denies:        p.Denies,
			Level:         p.Level,
		}
		if item.AccountID != nil && (*item.AccountID == 0 || *item.AccountID == 4294967295) {
			item.AccountID = nil
		}
		if item.Name != nil && strings.TrimSpace(*item.Name) == "" {
			item.Name = nil
		}
		if item.HeroID != nil && *item.HeroID == 0 {
			item.HeroID = nil
		}
		if p.PlayerSlot != nil {
			team := "radiant"
			if *p.PlayerSlot >= 128 {
				team = "dire"
			}
			item.Team = &team
			if data.RadiantWin != nil {
				won := (*p.PlayerSlot < 128) == *data.RadiantWin
				item.Won = &won
			}
		}
		result.Players = append(result.Players, item)
	}
	return result, nil
}
