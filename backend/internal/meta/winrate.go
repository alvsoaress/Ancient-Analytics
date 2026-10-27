package meta

import "fmt"

func WinRate(wins, picks int) (*float64, error) {
	if wins < 0 || picks < 0 || wins > picks {
		return nil, fmt.Errorf(
			"estatísticas inválidas: vitórias=%d, escolhas=%d",
			wins,
			picks,
		)
	}

	if picks == 0 {
		return nil, nil
	}

	rate := float64(wins) / float64(picks) * 100

	return &rate, nil
}
