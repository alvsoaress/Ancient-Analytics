# Deploy inicial

1. Copie `.env.example` para `.env` e defina `DOMAIN` com o domínio da API.
2. Crie um projeto no Supabase e copie a **Session pooler connection string** em **Project Settings → Database → Connect** para `DATABASE_URL`. Mantenha `sslmode=require`.
3. Confirme que a senha do banco está codificada para URL caso contenha caracteres especiais. O `.env` não deve ir para o Git.
4. Aponte o DNS desse domínio para a VPS e libere as portas 80 e 443.
5. Execute `docker compose up -d --build` na raiz do repositório. O serviço `migrate` aplica as migrações antes do User Service iniciar.

O Caddy é o único componente exposto publicamente. Gateway, Meta Service e Draft Service comunicam-se pela rede interna do Compose.

## Verificação

```bash
curl https://SEU_DOMINIO/api/status
curl -X POST https://SEU_DOMINIO/api/draft/sessions
```

As sessões de draft são mantidas em memória nesta primeira etapa e, portanto, o Draft Service deve operar com uma réplica. O User Service usa PostgreSQL/Supabase para persistir a identidade Steam verificada.
