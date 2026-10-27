package players

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/alvsoaress/ancient-analytics-backend/internal/opendota"
)

var (
	ErrInvalidAccountID = errors.New("accountId inválido")
	ErrNotFound         = errors.New("perfil não disponível na OpenDota")
)

type PlayerClient interface {
	Player(ctx context.Context, accountID uint32) (opendota.PlayerData, error)
}

type Profile struct {
	AccountID       uint32  `json:"accountId"`
	SteamID         *string `json:"steamId"`
	Name            *string `json:"name"`
	AvatarURL       *string `json:"avatarUrl"`
	ProfileURL      *string `json:"profileUrl"`
	RankTier        *int    `json:"rankTier"`
	LeaderboardRank *int    `json:"leaderboardRank"`
}

type Service struct {
	client PlayerClient
}

func NewService(client PlayerClient) *Service {
	return &Service{client: client}
}

func (s *Service) Profile(ctx context.Context, accountID uint32) (Profile, error) {
	if accountID == 0 {
		return Profile{}, ErrInvalidAccountID
	}
	data, err := s.client.Player(ctx, accountID)
	if errors.Is(err, opendota.ErrPlayerNotFound) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, fmt.Errorf("consultar perfil do jogador: %w", err)
	}
	if data.Profile == nil {
		return Profile{}, ErrNotFound
	}
	identity := data.Profile
	if identity.AccountID != accountID {
		return Profile{}, fmt.Errorf("perfil retornado não corresponde ao accountId solicitado")
	}
	profile := Profile{
		AccountID:       accountID,
		SteamID:         optionalString(identity.SteamID),
		Name:            optionalString(identity.PersonaName),
		ProfileURL:      steamProfileURL(identity.ProfileURL),
		RankTier:        availableRank(data.RankTier),
		LeaderboardRank: availableRank(data.LeaderboardRank),
	}
	for _, avatar := range []string{identity.AvatarFull, identity.AvatarMedium, identity.Avatar} {
		if value := httpsURL(avatar); value != nil {
			profile.AvatarURL = value
			break
		}
	}
	return profile, nil
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func availableRank(value *int) *int {
	if value == nil || *value <= 0 {
		return nil
	}
	rank := *value
	return &rank
}

func httpsURL(value string) *string {
	parsed, err := url.Parse(value)
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") || parsed.Hostname() == "" || parsed.User != nil {
		return nil
	}
	parsed.Scheme = "https"
	result := parsed.String()
	return &result
}

func steamProfileURL(value string) *string {
	parsed, err := url.Parse(value)
	if err != nil || !strings.EqualFold(parsed.Hostname(), "steamcommunity.com") {
		return nil
	}
	if parsed.Scheme == "http" {
		parsed.Scheme = "https"
	}
	return httpsURL(parsed.String())
}
