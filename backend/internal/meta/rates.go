package meta

// PickRate retorna a porcentagem de partidas em que o herói foi escolhido.
// Sem partidas conhecidas, a taxa é indisponível (nil), e não zero.
func PickRate(picks int, matches float64) *float64 {
	if matches <= 0 {
		return nil
	}

	rate := float64(picks) / matches * 100

	return &rate
}

// KDA retorna (abates + assistências) / mortes somados nas escolhas com K/D/A conhecidos.
// Sem mortes, divide por 1, como é usual no jogo. Sem escolhas com dados, retorna nil.
func KDA(kills, deaths, assists, picks int) *float64 {
	if picks <= 0 {
		return nil
	}

	ratio := float64(kills+assists) / float64(max(deaths, 1))

	return &ratio
}
