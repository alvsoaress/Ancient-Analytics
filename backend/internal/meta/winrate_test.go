package meta

import "testing"

func TestWinRate(t *testing.T) {
	rate, err := WinRate(3, 8)
	if err != nil {
		t.Fatalf("erro ao calcular taxa: %v", err)
	}

	if rate == nil {
		t.Fatal("esperada uma taxa para uma amostra com escolhas")
	}

	if *rate != 37.5 {
		t.Fatalf("taxa recebida: %v; esperado: 37.5", *rate)
	}
}

func TestWinRateWithoutPicks(t *testing.T) {
	rate, err := WinRate(0, 0)
	if err != nil {
		t.Fatalf("ausência de amostra não deveria gerar erro: %v", err)
	}

	if rate != nil {
		t.Fatalf("esperada taxa indisponível; recebido: %v", *rate)
	}
}

func TestWinRateWithNoWins(t *testing.T) {
	rate, err := WinRate(0, 10)
	if err != nil {
		t.Fatalf("erro ao calcular taxa: %v", err)
	}

	if rate == nil {
		t.Fatal("esperada taxa disponível, pois existem escolhas")
	}

	if *rate != 0 {
		t.Fatalf("taxa recebida: %v; esperado: 0", *rate)
	}
}

func TestWinRateRejectsInvalidStats(t *testing.T) {
	tests := []struct {
		name  string
		wins  int
		picks int
	}{
		{
			name:  "vitórias_negativas",
			wins:  -1,
			picks: 10,
		},
		{
			name:  "escolhas_negativas",
			wins:  0,
			picks: -1,
		},
		{
			name:  "vitórias_acima_das_escolhas",
			wins:  11,
			picks: 10,
		},
		{
			name:  "vitória_sem_escolhas",
			wins:  1,
			picks: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rate, err := WinRate(test.wins, test.picks)

			if err == nil {
				t.Fatal("esperado erro para estatísticas inválidas")
			}

			if rate != nil {
				t.Error("não deveria retornar taxa para dados inválidos")
			}
		})
	}
}
