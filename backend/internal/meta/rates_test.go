package meta

import "testing"

func TestPickRate(t *testing.T) {
	if rate := PickRate(5, 20); rate == nil || *rate != 25 {
		t.Fatalf("taxa recebida: %v; esperado: 25", rate)
	}
	if rate := PickRate(0, 20); rate == nil || *rate != 0 {
		t.Fatalf("zero escolhas deve resultar em 0%%: %v", rate)
	}
	if rate := PickRate(3, 0); rate != nil {
		t.Fatalf("sem partidas, a taxa deve ser indisponível: %v", *rate)
	}
}

func TestKDA(t *testing.T) {
	if ratio := KDA(10, 4, 6, 2); ratio == nil || *ratio != 4 {
		t.Fatalf("KDA recebido: %v; esperado: 4", ratio)
	}
	if ratio := KDA(3, 0, 5, 1); ratio == nil || *ratio != 8 {
		t.Fatalf("sem mortes, o KDA deve dividir por 1: %v", ratio)
	}
	if ratio := KDA(0, 0, 0, 0); ratio != nil {
		t.Fatalf("sem escolhas com dados, o KDA deve ser indisponível: %v", *ratio)
	}
}
