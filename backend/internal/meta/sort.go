package meta

import (
	"cmp"
	"fmt"
	"slices"
)

func sortHeroes(heroes []HeroPerformance, sortBy string) error {
	switch sortBy {
	case "":
		return nil
	case "winRate":
		slices.SortFunc(heroes, func(a, b HeroPerformance) int {
			switch {
			case a.WinRate == nil && b.WinRate != nil:
				return 1
			case a.WinRate != nil && b.WinRate == nil:
				return -1
			case a.WinRate != nil && b.WinRate != nil:
				if result := cmp.Compare(*b.WinRate, *a.WinRate); result != 0 {
					return result
				}
			}
			return cmp.Compare(a.HeroID, b.HeroID)
		})
	case "picks":
		slices.SortFunc(heroes, func(a, b HeroPerformance) int {
			if result := cmp.Compare(b.Picks, a.Picks); result != 0 {
				return result
			}
			return cmp.Compare(a.HeroID, b.HeroID)
		})
	default:
		return fmt.Errorf("critério de ordenação inválido: %q", sortBy)
	}
	return nil
}
