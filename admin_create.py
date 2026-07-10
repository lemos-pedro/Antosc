#!/usr/bin/env python3
"""
create_admin_user.py

Gera o hash bcrypt de uma password e produz o SQL de insert para criar
um utilizador admin diretamente na tabela `users` do towercore.

IMPORTANTE: ajusta TABLE_NAME e os nomes de coluna abaixo conforme o schema
real definido na migration 00002_users_auth. Os nomes usados aqui sao a
suposicao mais provavel dado o padrao do projeto (JWT + API Key, RBAC).

Uso:
    python create_admin_user.py
    (vai pedir username/email e password de forma interativa, sem ecoar a password)
"""

import getpass
import uuid
import sys
import bcrypt

TABLE_NAME = "users"

# Ajusta conforme o schema real (migration 00002_users_auth)
COLUMNS = {
    "id": "id",
    "username": "username",
    "email": "email",      # ou "email", conforme o schema
    "password_hash": "password_hash",
    "role": "role",
    "created_at": "created_at",
}

ROLE = "admin"  # ou "diretor_om" / "director" conforme o RBAC real definido


def main():
    print("=== Criar utilizador admin (towercore) ===\n")
    username = input("Username ou email: ").strip()
    if not username:
        print("Username/email obrigatorio.", file=sys.stderr)
        sys.exit(1)

    password = getpass.getpass("Password: ")
    password_confirm = getpass.getpass("Confirmar password: ")

    if password != password_confirm:
        print("Passwords nao coincidem.", file=sys.stderr)
        sys.exit(1)

    if len(password) < 8:
        print("AVISO: password com menos de 8 caracteres, considera algo mais forte.", file=sys.stderr)

    user_id = str(uuid.uuid4())
    hashed = bcrypt.hashpw(password.encode("utf-8"), bcrypt.gensalt(rounds=12))
    hashed_str = hashed.decode("utf-8")

    sql = f"""
INSERT INTO {TABLE_NAME} (
    {COLUMNS['id']},
    {COLUMNS['username']},
    {COLUMNS['email']},
    {COLUMNS['password_hash']},
    {COLUMNS['role']},
    {COLUMNS['created_at']}
) VALUES (
    '{user_id}',
    '{username}',
    '{hashed_str}',
    '{ROLE}',
    now()
);
""".strip()

    print("\n--- SQL gerado (revê antes de correr) ---\n")
    print(sql)
    print("\n--- Como correr ---")
    print("psql -h localhost -U A.lemos -d towercore")
    print("depois cola o SQL acima, ou guarda num ficheiro .sql e corre:")
    print('  psql -h localhost -U A.lemos -d towercore -f criar_admin.sql')
    print(f"\nuser_id gerado: {user_id}")
    print("Guarda este user_id se precisares de o referenciar manualmente depois.")


if __name__ == "__main__":
    main()