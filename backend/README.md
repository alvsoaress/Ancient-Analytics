# Backend — Ancient Analytics

Monólito modular em Go, usando apenas a biblioteca padrão. Consulta a [OpenDota](https://docs.opendota.com/), guarda as respostas em cache na memória e entrega ao frontend dados já calculados, como taxas de vitória e de escolha, KDA e posições estimadas. Não usa banco de dados nem Docker; persistência será adicionada quando houver dados próprios a guardar.

Requisito: Go 1.26 ou superior.

Na pasta `backend`, execute:

```bash
go run ./cmd/api
```

A API escuta apenas nesta máquina, em `http://127.0.0.1:8080`.

```bash
curl http://127.0.0.1:8080/api/health
```

Resposta: `{"status":"ok"}`. Essa rota verifica apenas se o servidor responde, sem consultar serviços externos.

## Estatísticas profissionais dos heróis

`GET /api/meta/heroes` retorna as estatísticas com `scope: "professional"`.
O parâmetro opcional `minPicks` inclui apenas heróis com quantidade de escolhas maior ou igual ao mínimo informado:

```bash
curl 'http://127.0.0.1:8080/api/meta/heroes?minPicks=10'
```

Sem o parâmetro, ou com `minPicks=0`, a consulta mantém todos os resultados.
Valores negativos, vazios, não inteiros, repetidos ou fora do limite de `int` retornam HTTP `400` com código `INVALID_MIN_PICKS`, antes de consultar o serviço.

O parâmetro opcional `sortBy` ordena os resultados de forma decrescente por taxa de vitória (`winRate`) ou quantidade de escolhas (`picks`):

```bash
curl 'http://127.0.0.1:8080/api/meta/heroes?minPicks=10&sortBy=winRate'
curl 'http://127.0.0.1:8080/api/meta/heroes?minPicks=10&sortBy=picks'
```

Empates são resolvidos pelo menor ID do herói. Na ordenação por taxa, valores indisponíveis (`null`) ficam no final, depois de taxas iguais a zero.
Sem `sortBy`, ou com o valor vazio, a consulta preserva a ordem dos dados recebidos. Critérios desconhecidos ou repetidos retornam HTTP `400` com código `INVALID_SORT_BY`, antes de consultar o serviço.
O filtro e a ordenação são aplicados aos resultados da consulta, sem alterar as estatísticas guardadas no cache.

Cada herói contém `heroId`, `name`, `picks`, `wins`, `winRate`, `pickRate` e `kda`:

- `winRate`: vitórias ÷ escolhas, em porcentagem; `null` sem escolhas.
- `pickRate`: porcentagem de partidas em que o herói foi escolhido. Sem `position`, a OpenDota não informa o total de partidas profissionais; ele é **estimado** como a soma das escolhas de todos os heróis dividida por dez (dez escolhas por partida), antes do filtro `minPicks`. Com `position`, usa o número exato de partidas incluídas na amostra. `null` quando não há partidas.
- `kda`: (abates + assistências) ÷ mortes, somados nas escolhas da amostra por posição; sem mortes, divide por 1. Sem `position`, é sempre `null`, pois `/heroStats` não fornece abates, mortes e assistências.

## Desempenho por posição

Use o parâmetro opcional `position` com um dos valores:

| Valor | Posição |
|---|---|
| `carry` | HC / posição 1 |
| `mid` | Mid / posição 2 |
| `offlane` | Offlane / posição 3 |
| `support4` | Suporte / posição 4 |
| `support5` | Suporte / posição 5 |

```bash
curl 'http://127.0.0.1:8080/api/meta/heroes?position=offlane&minPicks=1&sortBy=winRate'
curl 'http://127.0.0.1:8080/api/meta/heroes?position=support4&sortBy=picks'
curl 'http://127.0.0.1:8080/api/meta/heroes?position=support5&sortBy=picks'
```

Com `position`, escolhas e vitórias são agregadas exclusivamente naquela posição, a partir de uma amostra das até 20 partidas profissionais mais recentes listadas por `/proMatches` da OpenDota. As posições são **estimadas pela OpenDota**, usando o campo `position_est` dos detalhes de cada partida, inclusive para distinguir suportes 4 e 5. Não são deduzidas a partir do catálogo `Carry`/`Support`, nem do campo `lane_role`.

Somente partidas com resultado disponível e as cinco posições válidas, sem duplicação, em cada time entram na amostra. Partidas indisponíveis e partidas sem posições completas são excluídas e contabilizadas separadamente. Heróis sem escolhas naquela posição não aparecem na amostra. Um filtro alto, como `minPicks=10`, pode produzir uma lista vazia; comece com `minPicks=1`.

A resposta mantém `scope: "professional"` e acrescenta:

- `position`: posição consultada.
- `positionSource`: `opendota.position_est`, identificando a origem da estimativa.
- `sample`: limite, quantidade de partidas solicitadas/incluídas/indisponíveis/sem posições, IDs das partidas incluídas, menor e maior instante de início (`from` e `to`) e instante de coleta (`fetchedAt`).

O período é o das partidas efetivamente incluídas. Não representa um patch específico nem uma janela fixa de dias. **Este recorte por posição não é o mesmo agregado geral de `/heroStats`**: sem `position`, a rota mantém o comportamento anterior e usa os totais profissionais disponibilizados pela OpenDota.

A coleta usa no máximo quatro consultas de partidas simultâneas e um prazo total de 30 segundos. A primeira consulta com cache vazio faz até 22 chamadas externas (lista de partidas, até 20 detalhes e catálogo de heróis). A amostra completa das cinco posições é compartilhada em memória por cinco minutos; consultar outra posição reutiliza a mesma amostra. Reiniciar o backend limpa esse cache, assim como os demais caches em memória.

Posições desconhecidas, vazias ou repetidas retornam HTTP `400` com código `INVALID_POSITION`, sem consultar a fonte. Quando não há nenhuma partida utilizável, ou a coleta falha, a rota retorna HTTP `502` com código `POSITION_DATA_UNAVAILABLE`; ela não substitui esses resultados pelas taxas gerais dos heróis. Se apenas parte da coleta for utilizável, o sucesso inclui as contagens de exclusão em `sample`.

Referências da fonte: [OpenDota API](https://docs.opendota.com/), [estimativa de posições nos detalhes das partidas](https://github.com/odota/core/blob/master/svc/util/buildMatch.ts) e [campo `position_est`](https://github.com/odota/core/blob/master/global.d.ts).

## Perfil público do jogador

`GET /api/players/{accountId}` consulta os dados públicos disponíveis na OpenDota, sem login com Steam:

```bash
curl 'http://127.0.0.1:8080/api/players/86745912'
```

Use o `accountId` do Dota 2 (Steam32 / ID de amigo), um inteiro de 1 a 4294967295. O SteamID64 não é aceito neste endereço.

A resposta contém:

| Campo | Conteúdo |
|---|---|
| `accountId` | Conta consultada |
| `steamId` | SteamID64 como texto, para preservar todos os dígitos no JavaScript |
| `name` | Apelido público |
| `avatarUrl` | Avatar HTTPS disponível, preferindo a imagem maior |
| `profileUrl` | Link HTTPS para o perfil na comunidade Steam |
| `rankTier` | Código de medalha disponibilizado pela OpenDota |
| `leaderboardRank` | Colocação no ranking, quando disponível |

Os campos opcionais retornam `null` quando indisponíveis. Um código de rank ausente ou zero também retorna `null`; esta etapa não estima MMR, nem inclui histórico de partidas. Links inválidos são omitidos como `null`.

Respostas de erro seguem o formato `{"error":{"code":"...","message":"..."}}`:

- HTTP `400`, `INVALID_ACCOUNT_ID`: identificador inválido; não consulta a fonte.
- HTTP `404`, `PLAYER_NOT_FOUND`: perfil não disponível na OpenDota. Isso não afirma que a conta não existe nem identifica a causa da indisponibilidade.
- HTTP `502`, `UPSTREAM_UNAVAILABLE`: falha de consulta ou resposta inválida da fonte.

Perfis consultados com sucesso ficam em memória por cinco minutos, com até 128 contas guardadas. Consultas simultâneas ao mesmo perfil reutilizam o resultado após a primeira consulta. Erros não preenchem o cache; reiniciar o backend o limpa. Não há necessidade de banco de dados, chave Steam ou autenticação para esta funcionalidade.

Referência: [consulta de jogadores na OpenDota](https://docs.opendota.com/#tag/players).

## Histórico de partidas do jogador

`GET /api/players/{accountId}/matches` consulta o histórico público disponível na OpenDota, sem exigir login ou consultar previamente o perfil.

```bash
# Primeira página: até 20 partidas
curl 'http://127.0.0.1:8080/api/players/86745912/matches?limit=20&offset=0'

# Próxima página
curl 'http://127.0.0.1:8080/api/players/86745912/matches?limit=20&offset=20'
```

`limit` é a quantidade máxima de partidas da página: padrão 20, de 1 a 100. `offset` é quantas partidas pular: padrão 0, de 0 a 2147483546. Parâmetros vazios, repetidos, negativos, não inteiros ou fora desses limites retornam HTTP `400`, antes de acessar a fonte. Os códigos são `INVALID_ACCOUNT_ID`, `INVALID_LIMIT` e `INVALID_OFFSET`; uma query malformada retorna `INVALID_PAGINATION`.

A resposta usa este formato (exemplo ilustrativo):

```json
{
  "accountId": 86745912,
  "matches": [
    {
      "matchId": 9025128045,
      "heroId": 53,
      "won": true,
      "durationSeconds": 2375,
      "startTime": 1790890789,
      "kills": 13,
      "deaths": 2,
      "assists": 16,
      "gameMode": 22,
      "lobbyType": 7
    }
  ],
  "pagination": {
    "limit": 1,
    "offset": 0,
    "count": 1,
    "hasMore": true,
    "nextOffset": 1
  }
}
```

As partidas seguem a ordem decrescente de `matchId` fornecida pela OpenDota. A consulta inclui os modos disponíveis na fonte usando `significant=0`. `heroId`, `gameMode` e `lobbyType` são os identificadores da fonte; `startTime` é um instante Unix em segundos, e `durationSeconds` é a duração em segundos. `won` indica o resultado do jogador, combinando seu time com o time vencedor. Campos ausentes permanecem `null`, inclusive o resultado quando não há dados suficientes; zero abates continua sendo zero. Herói zero é tratado como indisponível.

Cada página faz **uma chamada externa**, pedindo até `limit + 1` partidas: a partida extra só informa `hasMore` e não aparece na lista. Para avançar, mantenha `limit` e use `nextOffset`. Na última página, `hasMore` é `false` e `nextOffset` é `null`, mesmo quando a quantidade de partidas é exatamente igual ao limite. Não é feita uma consulta adicional para contar o histórico inteiro.

Quando a fonte retorna uma lista vazia, a API responde HTTP `200`, `matches: []`, `count: 0` e encerra a paginação. Isso não distingue uma conta sem partidas de uma conta sem histórico público disponível. Falhas HTTP da fonte, timeout ou resposta inválida retornam HTTP `502`, `UPSTREAM_UNAVAILABLE`, sem substituir o erro por uma lista vazia.

Nesta etapa, o histórico é consultado diretamente, sem cache próprio ou banco. A paginação por deslocamento reflete os dados disponíveis em cada consulta: se novas partidas aparecerem entre páginas, podem ocorrer repetições ou mudanças de posição. Não é um retrato fixo do histórico completo.

Referências: [parâmetros de paginação e ordenação](https://github.com/odota/core/blob/master/svc/api/requests/playerParams.ts) e [campos do histórico](https://github.com/odota/core/blob/master/svc/api/responses/PlayerMatchesResponse.ts).

## Detalhes de uma partida

`GET /api/matches/{matchId}` consulta uma partida pública da OpenDota, sem login. O `matchId` pode ser obtido no histórico do jogador:

```bash
curl 'http://127.0.0.1:8080/api/matches/9025128045'
```

A resposta contém `matchId`, `durationSeconds`, `startTime` (Unix em segundos), `gameMode`, `lobbyType`, `winner`, `radiantScore`, `direScore` e `players`.

- `winner`: `radiant`, `dire` ou `null` quando o resultado não estiver disponível. Não é deduzido pelo placar.
- Cada jogador contém `accountId`, `name` (apelido público), `heroId`, `playerSlot`, `team`, `won`, `kills`, `deaths`, `assists`, `goldPerMinute`, `xpPerMinute`, `heroDamage`, `towerDamage`, `heroHealing`, `lastHits`, `denies` e `level`.
- `team` é `radiant` para slots 0–127 e `dire` para 128–255. Sem slot, `team` e `won` são `null`; sem vencedor, `won` é `null` mesmo que o time seja conhecido.
- Identificação de conta ausente, zero ou anônima (`4294967295`) retorna `accountId: null`. Nome vazio e herói zero também retornam `null`. Não são feitas consultas adicionais para identificar jogadores anônimos.
- Dados não fornecidos permanecem `null`, incluindo estatísticas dependentes de análise do replay. Zero abates, dano ou cura continuam sendo zero. Quando não há jogadores disponíveis, `players` é `[]`. A lista mantém a ordem da fonte e não exige exatamente dez jogadores, permitindo modos diferentes.

`matchId` deve ser um inteiro positivo até 9007199254740991, preservando a exatidão dos números no JavaScript. Identificadores inválidos retornam HTTP `400`, `INVALID_MATCH_ID`, sem consultar a fonte. Uma resposta HTTP `404` da OpenDota retorna HTTP `404`, `MATCH_NOT_FOUND`. Outras falhas de consulta ou respostas inconsistentes retornam HTTP `502`, `UPSTREAM_UNAVAILABLE`, sem expor detalhes internos.

A consulta faz uma chamada externa, com o timeout de dez segundos do cliente e limite de resposta de 16 MiB. Esta etapa não guarda detalhes em cache ou banco e não solicita processamento de replay; ela exibe os dados já disponíveis. O conteúdo publicado é limitado aos campos descritos acima.

Referência: [campos dos detalhes de partidas na OpenDota](https://github.com/odota/core/blob/master/svc/api/responses/MatchResponse.ts).

Para executar os testes e as verificações estáticas:

```bash
go test ./...
go test -race ./...
go vet ./...
```

## Organização

```text
backend/
├── cmd/api/main.go       # Monta as dependências e inicia o servidor
├── internal/api/         # Rotas, validação HTTP e respostas JSON
├── internal/meta/        # Cálculos, filtros e ordenação de desempenho
├── internal/matches/     # Detalhes, times e desempenho por partida
├── internal/opendota/    # Consulta à fonte externa e caches em memória
├── internal/players/     # Perfil, histórico paginado e dados opcionais
└── go.mod
```

Cada módulo mantém seus testes junto aos arquivos de implementação. O servidor em `cmd/api` monta os serviços de meta, jogadores e partidas usando o cliente OpenDota e os entrega ao roteador HTTP.
