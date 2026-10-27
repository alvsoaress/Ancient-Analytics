package meta

import "testing"

func TestSortHeroesByWinRate(t *testing.T) {
	rate50 := 50.0
	rate90 := 90.0

	heroes := []HeroPerformance{
		{HeroID: 3, WinRate: nil},
		{HeroID: 2, WinRate: &rate50},
		{HeroID: 4, WinRate: &rate90},
		{HeroID: 1, WinRate: &rate90},
	}

	if err := sortHeroes(heroes, "winRate"); err != nil {
		t.Fatalf("erro ao ordenar: %v", err)
	}

	expectedIDs := []int{1, 4, 2, 3}
	for i, expectedID := range expectedIDs {
		if heroes[i].HeroID != expectedID {
			t.Errorf("posição %d: ID recebido %d; esperado: %d", i, heroes[i].HeroID, expectedID)
		}
	}
}

func TestSortHeroesOptions(t *testing.T) {
	zero := 0.0
	tests := []struct {
		name    string
		sortBy  string
		heroes  []HeroPerformance
		wantIDs []int
		wantErr bool
	}{
		{
			name:   "escolhas_decrescentes_com_desempate",
			sortBy: "picks",
			heroes: []HeroPerformance{
				{HeroID: 3, Picks: 5},
				{HeroID: 2, Picks: 10},
				{HeroID: 1, Picks: 10},
			},
			wantIDs: []int{1, 2, 3},
		},
		{
			name:    "sem_criterio_preserva_ordem",
			heroes:  []HeroPerformance{{HeroID: 3}, {HeroID: 1}},
			wantIDs: []int{3, 1},
		},
		{
			name:    "criterio_desconhecido_nao_altera_lista",
			sortBy:  "unknown",
			heroes:  []HeroPerformance{{HeroID: 3}, {HeroID: 1}},
			wantIDs: []int{3, 1},
			wantErr: true,
		},
		{
			name:   "taxa_zero_antes_de_indisponivel",
			sortBy: "winRate",
			heroes: []HeroPerformance{
				{HeroID: 1, WinRate: nil},
				{HeroID: 2, WinRate: &zero},
			},
			wantIDs: []int{2, 1},
		},
		{
			name:    "taxas_indisponiveis_desempatam_por_id",
			sortBy:  "winRate",
			heroes:  []HeroPerformance{{HeroID: 3}, {HeroID: 1}},
			wantIDs: []int{1, 3},
		},
		{name: "lista_vazia_por_taxa", sortBy: "winRate"},
		{name: "lista_vazia_por_escolhas", sortBy: "picks"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := sortHeroes(test.heroes, test.sortBy)
			if (err != nil) != test.wantErr {
				t.Fatalf("erro recebido: %v; esperado erro: %t", err, test.wantErr)
			}
			if len(test.heroes) != len(test.wantIDs) {
				t.Fatalf("quantidade de resultados: %d; esperado: %d", len(test.heroes), len(test.wantIDs))
			}
			for i, wantID := range test.wantIDs {
				if test.heroes[i].HeroID != wantID {
					t.Errorf("posição %d: ID recebido %d; esperado: %d", i, test.heroes[i].HeroID, wantID)
				}
			}
		})
	}
}
