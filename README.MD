# Ancient Analytics

Aplicação web local para consultar estatísticas de Dota 2 por meio de APIs externas.

## Escopo

A primeira versão terá meta, heróis, jogadores e partidas, com consulta pública. Personalização fica para depois. O backend já consulta o catálogo de heróis, estatísticas de meta e o perfil público de jogadores. Histórico de partidas e demais consultas de partidas seguem planejados.

## Arquitetura

- Frontend: React, TypeScript e Vite. Atualmente contém a tela inicial do Vite.
- Backend: monólito modular em Go, usando a biblioteca padrão.
- Integração: OpenDota, acessada pelo backend com cache em memória.
- Persistência: será adicionada quando houver dados próprios a guardar. O backend inicial não depende de banco nem Docker.

## Backend

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

### Estatísticas profissionais dos heróis

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

### Desempenho por posição

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

### Perfil público do jogador

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

Para executar os testes e as verificações estáticas:

```bash
go test ./...
go test -race ./...
go vet ./...
```

### Organização

```text
backend/
├── cmd/api/main.go       # Monta as dependências e inicia o servidor
├── internal/api/         # Rotas, validação HTTP e respostas JSON
├── internal/meta/        # Cálculos, filtros e ordenação de desempenho
├── internal/opendota/    # Consulta à fonte externa e caches em memória
├── internal/players/     # Perfil público e tratamento de dados opcionais
└── go.mod
```

Cada módulo mantém seus testes junto aos arquivos de implementação. O módulo de partidas será adicionado quando sua funcionalidade for implementada.

## Frontend

Na pasta `frontend`, instale as dependências e inicie o servidor de desenvolvimento:

```bash
npm install
npm run dev
```

O frontend ainda não está conectado ao backend.
