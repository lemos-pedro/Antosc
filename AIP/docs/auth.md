# Autenticação e utilizadores

## Modelo

- Tabela `users` (migration `006_users.sql`)
- Password com bcrypt
- JWT HS256 (implementação interna, sem dependência extra)
- Roles alinhadas às personas + `admin`

Roles: `admin`, `om`, `engenharia`, `controller`, `financeiro`,
`diretor_tecnico`, `ceo`, `conselho_administracao`

## Endpoints

| Método | Path | Quem |
|--------|------|------|
| POST | `/api/v1/auth/bootstrap` | público, **só se 0 users** — cria 1.º admin |
| POST | `/api/v1/auth/login` | público |
| GET | `/api/v1/auth/me` | autenticado |
| POST | `/api/v1/auth/change-password` | autenticado |
| POST | `/api/v1/auth/users` | admin — criar user |
| GET | `/api/v1/auth/users` | admin — listar |
| PATCH | `/api/v1/auth/users/{id}` | admin — role / active |

## Bootstrap (primeiro admin)

```bash
curl -s -X POST localhost:8090/api/v1/auth/bootstrap \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@antosc.com","full_name":"Admin","password":"senha-forte-123"}'
```

Resposta: `{ "token": "...", "user": { "role": "admin", ... } }`

## Login

```bash
curl -s -X POST localhost:8090/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@antosc.com","password":"senha-forte-123"}'
```

## Criar utilizador (admin)

```bash
TOKEN=... # do login
curl -s -X POST localhost:8090/api/v1/auth/users \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"email":"om@antosc.com","full_name":"Técnico O&M","password":"senha-forte-123","role":"om"}'
```

## Política de acesso

| Área | Auth |
|------|------|
| health, login, bootstrap | público |
| mutações (norms, confirm, trigger predict) | JWT |
| assistente | JWT (role do token) |
| leituras / Power BI / exports | JWT **ou** `X-API-Key` |

O assistente, ao chamar tools, envia `X-API-Key` (valor de `AIP_API_KEY` ou, se vazio, o `AIP_JWT_SECRET`).

## Variáveis

```bash
AIP_JWT_SECRET=...     # obrigatório ≥32 chars
AIP_JWT_TTL_HOURS=24
AIP_API_KEY=...        # opcional; Power BI + tools
```

## RBAC (autorização por role)

`admin` tem acesso a **todas** as operações.

| Operação | Roles permitidas |
|----------|------------------|
| Criar/editar/desativar normas | `controller` |
| Confirmar causa de incidente | `om`, `diretor_tecnico` |
| Disparar previsão (`POST /predictions`) | `engenharia`, `diretor_tecnico` |
| Gestão de utilizadores | `admin` |
| Assistente | qualquer autenticado (role do token) |
| Leituras (incidentes, consumo, Power BI) | JWT de qualquer role **ou** `X-API-Key` |

Respostas:
- `401` — sem token / token inválido
- `403` — autenticado mas role insuficiente


## Audit de autenticação

Eventos gravados em `auth_audit_log` (migration 007):

- `login_success` / `login_failure`
- `bootstrap`
- `user_created`
- `password_change`

```bash
curl -s -H "Authorization: Bearer $ADMIN_TOKEN" localhost:8090/api/v1/auth/audit | jq .
```

Só **admin**.


## Refresh token + logout + lockout

Login / bootstrap devolvem:

```json
{
  "token": "<access JWT>",
  "refresh_token": "<opaco>",
  "expires_at": "...",
  "user": { }
}
```

| Endpoint | Auth | Função |
|----------|------|--------|
| `POST /api/v1/auth/refresh` | público + body refresh_token | novo access + refresh (rotação) |
| `POST /api/v1/auth/logout` | JWT + opcional refresh_token | revoga refresh(s) |

Lockout: após `AIP_MAX_FAILED_LOGINS` (default 5) falhas, conta bloqueada `AIP_LOCKOUT_MINUTES` (default 15).

Access token default: **1 hora**. Refresh: **7 dias**.
