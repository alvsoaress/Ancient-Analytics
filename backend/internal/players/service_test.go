package players

import (
	"context"
	"errors"
	"testing"

	"github.com/alvsoaress/ancient-analytics-backend/internal/opendota"
)

type fakePlayerClient struct {
	data       opendota.PlayerData
	err        error
	receivedID *uint32
}

func (f fakePlayerClient) Player(ctx context.Context, id uint32) (opendota.PlayerData, error) {
	if f.receivedID != nil {
		*f.receivedID = id
	}
	return f.data, f.err
}

func TestProfileMapsPublicFields(t *testing.T) {
	rank, leaderboard := 80, 123
	received := uint32(0)
	service := NewService(fakePlayerClient{receivedID: &received, data: opendota.PlayerData{
		Profile:  &opendota.PlayerIdentity{AccountID: 42, SteamID: "76561198000000000", PersonaName: "Player", AvatarFull: "https://avatars.steamstatic.com/full.jpg", ProfileURL: "http://steamcommunity.com/profiles/76561198000000000/"},
		RankTier: &rank, LeaderboardRank: &leaderboard,
	}})
	profile, err := service.Profile(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if received != 42 || profile.AccountID != 42 || profile.Name == nil || *profile.Name != "Player" || profile.SteamID == nil || *profile.SteamID != "76561198000000000" {
		t.Fatalf("identificação inesperada: %+v", profile)
	}
	if profile.AvatarURL == nil || *profile.AvatarURL != "https://avatars.steamstatic.com/full.jpg" || profile.ProfileURL == nil || *profile.ProfileURL != "https://steamcommunity.com/profiles/76561198000000000/" {
		t.Fatalf("links inesperados: %+v", profile)
	}
	if profile.RankTier == nil || *profile.RankTier != 80 || profile.LeaderboardRank == nil || *profile.LeaderboardRank != 123 {
		t.Fatalf("rank inesperado: %+v", profile)
	}
	*profile.RankTier = 1
	if rank != 80 {
		t.Fatal("a resposta não deve alterar os dados recebidos do cliente")
	}
}

func TestProfilePreservesUnavailableDataAndSafeLinks(t *testing.T) {
	zero := 0
	for _, rank := range []*int{nil, &zero} {
		service := NewService(fakePlayerClient{data: opendota.PlayerData{
			Profile:  &opendota.PlayerIdentity{AccountID: 42, AvatarFull: "javascript:alert(1)", AvatarMedium: "https://avatars.steamstatic.com/medium.jpg", ProfileURL: "https://example.com/pretend-steam"},
			RankTier: rank, LeaderboardRank: rank,
		}})
		profile, err := service.Profile(context.Background(), 42)
		if err != nil {
			t.Fatal(err)
		}
		if profile.RankTier != nil || profile.LeaderboardRank != nil || profile.SteamID != nil || profile.Name != nil || profile.ProfileURL != nil {
			t.Fatalf("dados ausentes não devem ser inventados: %+v", profile)
		}
		if profile.AvatarURL == nil || *profile.AvatarURL != "https://avatars.steamstatic.com/medium.jpg" {
			t.Fatal("deve usar o próximo avatar HTTPS disponível")
		}
	}
}

func TestProfileErrors(t *testing.T) {
	for _, test := range []struct {
		name   string
		client fakePlayerClient
		id     uint32
		want   error
	}{
		{"id_invalido", fakePlayerClient{}, 0, ErrInvalidAccountID},
		{"perfil_indisponivel", fakePlayerClient{err: opendota.ErrPlayerNotFound}, 42, ErrNotFound},
		{"sem_identificacao", fakePlayerClient{}, 42, ErrNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewService(test.client).Profile(context.Background(), test.id)
			if !errors.Is(err, test.want) {
				t.Fatalf("erro inesperado: %v; esperado: %v", err, test.want)
			}
		})
	}
	expected := errors.New("falha na fonte")
	if _, err := NewService(fakePlayerClient{err: expected}).Profile(context.Background(), 42); !errors.Is(err, expected) {
		t.Fatalf("causa do erro não preservada: %v", err)
	}
	if _, err := NewService(fakePlayerClient{data: opendota.PlayerData{Profile: &opendota.PlayerIdentity{AccountID: 43}}}).Profile(context.Background(), 42); err == nil {
		t.Fatal("conta incorreta aceita")
	}
}
