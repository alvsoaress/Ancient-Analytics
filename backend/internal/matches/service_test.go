package matches

import (
	"context"
	"errors"
	"testing"

	"github.com/alvsoaress/ancient-analytics-backend/internal/opendota"
)

type fakeMatchClient struct {
	data       opendota.MatchData
	err        error
	receivedID *int64
}

func (f fakeMatchClient) Match(ctx context.Context, id int64) (opendota.MatchData, error) {
	if f.receivedID != nil {
		*f.receivedID = id
	}
	return f.data, f.err
}

func TestDetailsMapsMatchAndPlayerPerformance(t *testing.T) {
	id := int64(9025128045)
	duration, mode, lobby, radiantScore, direScore := 2375, 22, 7, 43, 26
	start := int64(1790890789)
	winner := true
	account := uint32(42)
	name := "Player"
	slot, hero, kills, deaths, assists, gpm, xpm, damage, tower, healing, lh, denies, level := 0, 53, 0, 2, 16, 500, 600, 10000, 2000, 0, 150, 3, 20
	received := int64(0)
	client := fakeMatchClient{receivedID: &received, data: opendota.MatchData{MatchID: id, Duration: &duration, StartTime: &start, GameMode: &mode, LobbyType: &lobby, RadiantWin: &winner, RadiantScore: &radiantScore, DireScore: &direScore, Players: []*opendota.MatchPlayer{{AccountID: &account, PersonaName: &name, PlayerSlot: &slot, HeroID: &hero, Kills: &kills, Deaths: &deaths, Assists: &assists, GoldPerMinute: &gpm, XPPerMinute: &xpm, HeroDamage: &damage, TowerDamage: &tower, HeroHealing: &healing, LastHits: &lh, Denies: &denies, Level: &level}}}}
	result, err := NewService(client).Details(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if received != id || result.MatchID != id || result.DurationSeconds == nil || *result.DurationSeconds != duration || result.StartTime == nil || *result.StartTime != start || result.GameMode == nil || *result.GameMode != 22 || result.LobbyType == nil || *result.LobbyType != 7 || result.Winner == nil || *result.Winner != "radiant" || result.RadiantScore == nil || *result.RadiantScore != 43 || result.DireScore == nil || *result.DireScore != 26 || len(result.Players) != 1 {
		t.Fatalf("detalhes inesperados: %+v", result)
	}
	p := result.Players[0]
	if p.AccountID == nil || *p.AccountID != 42 || p.Name == nil || *p.Name != name || p.HeroID == nil || *p.HeroID != 53 || p.Team == nil || *p.Team != "radiant" || p.Won == nil || !*p.Won || p.PlayerSlot == nil || *p.PlayerSlot != 0 {
		t.Fatalf("identificação inesperada: %+v", p)
	}
	if p.Kills == nil || *p.Kills != 0 || p.Deaths == nil || *p.Deaths != 2 || p.Assists == nil || *p.Assists != 16 || p.GoldPerMinute == nil || *p.GoldPerMinute != 500 || p.XPPerMinute == nil || *p.XPPerMinute != 600 || p.HeroDamage == nil || *p.HeroDamage != 10000 || p.TowerDamage == nil || *p.TowerDamage != 2000 || p.HeroHealing == nil || *p.HeroHealing != 0 || p.LastHits == nil || *p.LastHits != 150 || p.Denies == nil || *p.Denies != 3 || p.Level == nil || *p.Level != 20 {
		t.Fatalf("desempenho inesperado: %+v", p)
	}
}

func TestDetailsDerivesBothTeamsWithoutInventingResults(t *testing.T) {
	for _, radiantWin := range []bool{true, false} {
		radiantSlot, direSlot := 0, 128
		data := opendota.MatchData{MatchID: 1, RadiantWin: &radiantWin, Players: []*opendota.MatchPlayer{{PlayerSlot: &radiantSlot}, {PlayerSlot: &direSlot}, {}}}
		result, err := NewService(fakeMatchClient{data: data}).Details(context.Background(), 1)
		if err != nil {
			t.Fatal(err)
		}
		wanted := "dire"
		if radiantWin {
			wanted = "radiant"
		}
		if result.Winner == nil || *result.Winner != wanted {
			t.Fatal("vencedor incorreto")
		}
		if *result.Players[0].Team != "radiant" || *result.Players[0].Won != radiantWin || *result.Players[1].Team != "dire" || *result.Players[1].Won == radiantWin {
			t.Fatal("resultado por time incorreto")
		}
		if result.Players[2].Team != nil || result.Players[2].Won != nil {
			t.Fatal("jogador sem slot deve manter time e resultado indisponíveis")
		}
	}
	slot, zero := 0, 0
	blank := " "
	account := uint32(4294967295)
	result, err := NewService(fakeMatchClient{data: opendota.MatchData{MatchID: 1, Players: []*opendota.MatchPlayer{{AccountID: &account, PersonaName: &blank, HeroID: &zero, PlayerSlot: &slot}}}}).Details(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	p := result.Players[0]
	if result.Winner != nil || p.Won != nil || p.HeroID != nil || p.AccountID != nil || p.Name != nil || p.Kills != nil || p.Team == nil {
		t.Fatalf("dados ausentes foram inventados: %+v", result)
	}
	empty, err := NewService(fakeMatchClient{data: opendota.MatchData{MatchID: 1}}).Details(context.Background(), 1)
	if err != nil || empty.Players == nil || len(empty.Players) != 0 {
		t.Fatal("jogadores ausentes devem gerar lista vazia")
	}
}

func TestDetailsValidatesIDAndMapsErrors(t *testing.T) {
	for _, id := range []int64{0, -1, 9007199254740992} {
		received := int64(0)
		_, err := NewService(fakeMatchClient{receivedID: &received}).Details(context.Background(), id)
		if !errors.Is(err, ErrInvalidMatchID) || received != 0 {
			t.Fatalf("ID inválido não deveria consultar a fonte: %v", err)
		}
	}
	if _, err := NewService(fakeMatchClient{err: opendota.ErrMatchNotFound}).Details(context.Background(), 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("partida indisponível não mapeada: %v", err)
	}
	expected := errors.New("falha da fonte")
	if _, err := NewService(fakeMatchClient{err: expected}).Details(context.Background(), 1); !errors.Is(err, expected) {
		t.Fatalf("causa não preservada: %v", err)
	}
	if _, err := NewService(fakeMatchClient{data: opendota.MatchData{MatchID: 2}}).Details(context.Background(), 1); err == nil {
		t.Fatal("partida incorreta aceita")
	}
}
