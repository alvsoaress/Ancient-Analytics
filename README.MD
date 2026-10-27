# Ancient Analytics

Aplicação web local para consultar estatísticas de Dota 2 por meio de APIs externas.

## Escopo

A primeira versão terá meta, heróis, jogadores e partidas, com consulta pública. Personalização fica para depois. As funcionalidades de consulta ainda serão implementadas.

## Arquitetura

- Frontend: React, TypeScript e Vite. Atualmente contém a tela inicial do Vite.
- Backend: monólito modular em Go, usando a biblioteca padrão.
- Integração planejada: OpenDota, acessada pelo backend.
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

Para executar os testes e as verificações estáticas:

```bash
go test ./...
go vet ./...
```

### Organização

```text
backend/
├── cmd/api/main.go          # Inicialização e encerramento do servidor
├── internal/api/router.go  # Registro das rotas HTTP
├── internal/api/router_test.go
└── go.mod
```

Os módulos de heróis, meta, jogadores e partidas serão criados conforme suas funcionalidades forem implementadas.

## Frontend

Na pasta `frontend`, instale as dependências e inicie o servidor de desenvolvimento:

```bash
npm install
npm run dev
```

O frontend ainda não está conectado ao backend.
