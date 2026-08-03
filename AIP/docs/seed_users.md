# Seed de utilizadores

Cria **admin** + um user por persona. Idempotente (não duplica emails).

```bash
# Depois de make migrate
make seed

# Ou com password/domínio custom
SEED_PASSWORD='SenhaForte!2026' SEED_EMAIL_DOMAIN=empresa.com make seed
```

## Contas criadas (domínio default `antosc.local`)

| Email | Role |
|-------|------|
| admin@antosc.local | admin |
| om@antosc.local | om |
| engenharia@antosc.local | engenharia |
| controller@antosc.local | controller |
| financeiro@antosc.local | financeiro |
| diretor.tecnico@antosc.local | diretor_tecnico |
| ceo@antosc.local | ceo |
| conselho@antosc.local | conselho_administracao |

Password inicial: `Trocar123!` (ou `SEED_PASSWORD`).

**Alterar após primeiro login:**

```bash
curl -X POST localhost:8090/api/v1/auth/change-password \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"current_password":"Trocar123!","new_password":"nova-senha-forte"}'
```

Nota: o seed **não** exige bootstrap. Se já correste bootstrap, o admin existente é mantido e os restantes roles são criados.
