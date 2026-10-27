package players

import (
	"context"
	"errors"
	"fmt"
)

const (
	DefaultMatchLimit = 20
	MaxMatchLimit     = 100
	// Reserva espaço para avançar uma página e para o item extra em inteiros de 32 bits.
	MaxMatchOffset = 2147483546
)

var ErrInvalidPagination = errors.New("paginação de partidas inválida")

type Match struct {
	MatchID         int64  `json:"matchId"`
	HeroID          *int   `json:"heroId"`
	Won             *bool  `json:"won"`
	DurationSeconds *int   `json:"durationSeconds"`
	StartTime       *int64 `json:"startTime"`
	Kills           *int   `json:"kills"`
	Deaths          *int   `json:"deaths"`
	Assists         *int   `json:"assists"`
	GameMode        *int   `json:"gameMode"`
	LobbyType       *int   `json:"lobbyType"`
}

type Pagination struct {
	Limit      int  `json:"limit"`
	Offset     int  `json:"offset"`
	Count      int  `json:"count"`
	HasMore    bool `json:"hasMore"`
	NextOffset *int `json:"nextOffset"`
}

type MatchHistory struct {
	AccountID  uint32     `json:"accountId"`
	Matches    []Match    `json:"matches"`
	Pagination Pagination `json:"pagination"`
}

func (s *Service) Matches(ctx context.Context, accountID uint32, limit, offset int) (MatchHistory, error) {
	if accountID == 0 {
		return MatchHistory{}, ErrInvalidAccountID
	}
	if limit < 1 || limit > MaxMatchLimit || offset < 0 || offset > MaxMatchOffset {
		return MatchHistory{}, ErrInvalidPagination
	}
	data, err := s.client.PlayerMatches(ctx, accountID, limit+1, offset)
	if err != nil {
		return MatchHistory{}, fmt.Errorf("consultar histórico do jogador: %w", err)
	}
	hasMore := len(data) > limit
	if hasMore {
		data = data[:limit]
	}
	result := MatchHistory{
		AccountID: accountID, Matches: make([]Match, 0, len(data)),
		Pagination: Pagination{Limit: limit, Offset: offset, Count: len(data), HasMore: hasMore},
	}
	if hasMore {
		next := offset + limit
		result.Pagination.NextOffset = &next
	}
	for _, match := range data {
		item := Match{
			MatchID: match.MatchID, HeroID: match.HeroID, DurationSeconds: match.Duration, StartTime: match.StartTime,
			Kills: match.Kills, Deaths: match.Deaths, Assists: match.Assists, GameMode: match.GameMode, LobbyType: match.LobbyType,
		}
		if item.HeroID != nil && *item.HeroID == 0 {
			item.HeroID = nil
		}
		if match.PlayerSlot != nil && match.RadiantWin != nil {
			won := (*match.PlayerSlot < 128) == *match.RadiantWin
			item.Won = &won
		}
		result.Matches = append(result.Matches, item)
	}
	return result, nil
}
