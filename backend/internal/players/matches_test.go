package players

import (
	"context"
	"errors"
	"testing"

	"github.com/alvsoaress/ancient-analytics-backend/internal/opendota"
)

func (f fakePlayerClient) PlayerMatches(context.Context, uint32, int, int) ([]opendota.PlayerMatch, error) {
	return nil, nil
}

type fakeHistoryClient struct {
	fakePlayerClient
	matches       []opendota.PlayerMatch
	err           error
	id            uint32
	limit, offset int
	calls         int
}

func (f *fakeHistoryClient) PlayerMatches(ctx context.Context, id uint32, limit, offset int) ([]opendota.PlayerMatch, error) {
	f.id, f.limit, f.offset = id, limit, offset
	f.calls++
	return f.matches, f.err
}

func TestMatchesPaginatesWithLookahead(t *testing.T) {
	for _, test := range []struct {
		name  string
		count int
		more  bool
	}{
		{"vazia", 0, false}, {"ultima_incompleta", 1, false}, {"ultima_completa", 2, false}, {"tem_proxima", 3, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := make([]opendota.PlayerMatch, test.count)
			for i := range data {
				data[i].MatchID = int64(100 - i)
			}
			client := &fakeHistoryClient{matches: data}
			page, err := NewService(client).Matches(context.Background(), 42, 2, 4)
			if err != nil {
				t.Fatal(err)
			}
			if client.id != 42 || client.limit != 3 || client.offset != 4 || client.calls != 1 {
				t.Fatalf("paginação não encaminhada: %+v", client)
			}
			count := min(test.count, 2)
			if page.AccountID != 42 || len(page.Matches) != count || page.Matches == nil || page.Pagination.Count != count || page.Pagination.Limit != 2 || page.Pagination.Offset != 4 || page.Pagination.HasMore != test.more {
				t.Fatalf("página inesperada: %+v", page)
			}
			if test.more {
				if page.Pagination.NextOffset == nil || *page.Pagination.NextOffset != 6 {
					t.Fatal("próximo offset deveria ser 6")
				}
			} else if page.Pagination.NextOffset != nil {
				t.Fatal("última página não tem próximo offset")
			}
			for i, m := range page.Matches {
				if m.MatchID != data[i].MatchID {
					t.Fatal("ordem da fonte deve ser preservada")
				}
			}
		})
	}
}

func TestMatchesCalculatesPlayerResultAndPreservesMissingData(t *testing.T) {
	for _, test := range []struct {
		slot             int
		radiantWin, want bool
	}{{0, true, true}, {4, false, false}, {128, false, true}, {132, true, false}} {
		client := &fakeHistoryClient{matches: []opendota.PlayerMatch{{MatchID: 1, PlayerSlot: &test.slot, RadiantWin: &test.radiantWin}}}
		page, err := NewService(client).Matches(context.Background(), 42, 20, 0)
		if err != nil || page.Matches[0].Won == nil || *page.Matches[0].Won != test.want {
			t.Fatalf("resultado incorreto: %+v; erro: %v", page, err)
		}
	}
	hero, kills, deaths, assists, duration, mode, lobby := 1, 0, 2, 5, 1800, 22, 7
	start := int64(1700000000)
	client := &fakeHistoryClient{matches: []opendota.PlayerMatch{{MatchID: 9025128045, HeroID: &hero, Kills: &kills, Deaths: &deaths, Assists: &assists, Duration: &duration, StartTime: &start, GameMode: &mode, LobbyType: &lobby}, {MatchID: 2}}}
	page, err := NewService(client).Matches(context.Background(), 42, 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	first := page.Matches[0]
	if first.HeroID == nil || *first.HeroID != hero || first.Kills == nil || *first.Kills != 0 || first.Deaths == nil || *first.Deaths != 2 || first.Assists == nil || *first.Assists != 5 || first.DurationSeconds == nil || *first.DurationSeconds != 1800 || first.StartTime == nil || *first.StartTime != start || first.GameMode == nil || *first.GameMode != 22 || first.LobbyType == nil || *first.LobbyType != 7 {
		t.Fatalf("campos não foram mapeados: %+v", first)
	}
	second := page.Matches[1]
	if second.Won != nil || second.HeroID != nil || second.Kills != nil || second.DurationSeconds != nil || second.StartTime != nil {
		t.Fatalf("dados ausentes não podem virar zero ou derrota: %+v", second)
	}
}

func TestMatchesValidatesInputsAndPreservesSourceErrors(t *testing.T) {
	client := &fakeHistoryClient{}
	for _, test := range []struct {
		id            uint32
		limit, offset int
		want          error
	}{{0, 20, 0, ErrInvalidAccountID}, {42, 0, 0, ErrInvalidPagination}, {42, 101, 0, ErrInvalidPagination}, {42, 20, -1, ErrInvalidPagination}, {42, 20, 2147483647, ErrInvalidPagination}} {
		if _, err := NewService(client).Matches(context.Background(), test.id, test.limit, test.offset); !errors.Is(err, test.want) {
			t.Fatalf("erro inesperado: %v", err)
		}
	}
	if client.calls != 0 {
		t.Fatal("entrada inválida não deve consultar a fonte")
	}
	expected := errors.New("fonte indisponível")
	client.err = expected
	if _, err := NewService(client).Matches(context.Background(), 42, 20, 0); !errors.Is(err, expected) {
		t.Fatalf("causa não preservada: %v", err)
	}
}
