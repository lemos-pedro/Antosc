// cmd/seed_users cria o admin e um utilizador por persona, se ainda não existirem.
//
// Uso:
//
//	go run ./cmd/seed_users
//	SEED_PASSWORD=Trocar123! go run ./cmd/seed_users
//
// Não sobrescreve emails já existentes. Seguro para correr várias vezes (idempotente).
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/antosc/aip/internal/auth"
	"github.com/antosc/aip/internal/config"
	"github.com/antosc/aip/internal/logger"
	"github.com/antosc/aip/internal/repository/postgres"
)

type seedUser struct {
	Email    string
	FullName string
	Role     string
}

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)

	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		password = "Trocar123!"
		log.Warn("SEED_PASSWORD não definido — a usar password default 'Trocar123!' (alterar após login)")
	}
	if len(password) < 8 {
		log.Error("SEED_PASSWORD deve ter pelo menos 8 caracteres")
		os.Exit(1)
	}

	domain := os.Getenv("SEED_EMAIL_DOMAIN")
	if domain == "" {
		domain = "antosc.co.ao"
	}

	db, err := postgres.New(cfg.Database.DSN())
	if err != nil {
		log.Error("ligação à base de dados falhou", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	users := postgres.NewUserRepository(db)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Error("erro ao gerar hash", "err", err)
		os.Exit(1)
	}

	seeds := []seedUser{
		{Email: "ariclene.lemos@" + domain, FullName: "Administrador AIP", Role: "admin"},
		{Email: "rene.lima@" + domain, FullName: "René Lima", Role: "om"},
		{Email: "evaristo.augusto@" + domain, FullName: "Evaristo Augusto", Role: "engenharia"},
		{Email: "alfredo.antunes@" + domain, FullName: "Alfredo Antunes", Role: "controller"},
		{Email: "alfredo.antunes@" + domain, FullName: "Financeiro", Role: "financeiro"},
		{Email: "joao.baptista@" + domain, FullName: "João Baptista", Role: "diretor_tecnico"},
		{Email: "ceo@" + domain, FullName: "CEO", Role: "ceo"},
		{Email: "conselho@" + domain, FullName: "Conselho de Administração", Role: "conselho_administracao"},
	}

	created, skipped, failed := 0, 0, 0
	for _, s := range seeds {
		email := strings.ToLower(s.Email)
		_, err := users.GetByEmail(ctx, email)
		if err == nil {
			log.Info("já existe — omitido", "email", email, "role", s.Role)
			skipped++
			continue
		}
		if err != postgres.ErrUserNotFound {
			log.Warn("erro ao verificar email", "email", email, "err", err)
			failed++
			continue
		}

		u, err := users.Create(ctx, postgres.User{
			Email:        email,
			FullName:     s.FullName,
			PasswordHash: hash,
			Role:         s.Role,
		})
		if err != nil {
			log.Warn("falha ao criar", "email", email, "err", err)
			failed++
			continue
		}
		created++
		log.Info("utilizador criado", "email", u.Email, "role", u.Role, "id", u.ID)
	}

	fmt.Println()
	fmt.Printf("Seed concluído: created=%d skipped=%d failed=%d\n", created, skipped, failed)
	fmt.Println("Password inicial (todos):", password)
	fmt.Println("IMPORTANTE: alterar passwords após o primeiro login (POST /api/v1/auth/change-password).")
	fmt.Println()
	fmt.Println("Login exemplo:")
	fmt.Printf("  curl -s -X POST localhost:8090/api/v1/auth/login -H 'Content-Type: application/json' \\\n")
	fmt.Printf("    -d '{\"email\":\"admin@%s\",\"password\":\"%s\"}'\n", domain, password)

	if failed > 0 {
		os.Exit(1)
	}
}
