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
