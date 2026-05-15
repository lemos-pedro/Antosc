.PHONY: build run test fmt migrate-up migrate-snmp migrate-audit seed-dev docker-up docker-down

BINARY=towercore
CMD=./cmd/api
GOCACHE?=/tmp/go-build
DB_URL?=postgres://postgres:postgres@localhost:5432/antosc?sslmode=disable

build:
	GOCACHE=$(GOCACHE) go build -o bin/$(BINARY) $(CMD)

run:
	GOCACHE=$(GOCACHE) go run $(CMD)

test:
	GOCACHE=$(GOCACHE) go test ./... -v

fmt:
	gofmt -w ./cmd ./internal ./pkg

migrate-up:
	psql "$(DB_URL)" -f ../sql/0.sql

migrate-snmp:
	psql "$(DB_URL)" -f ../sql/1_alter_towers_snmp.sql

migrate-audit:
	psql "$(DB_URL)" -f ../sql/2_audit_logs.sql

seed-dev:
	psql "$(DB_URL)" -f ../sql/seed_dev.sql

docker-up:
	docker compose -f ../docker-compose.yml up -d --build

docker-down:
	docker compose -f ../docker-compose.yml down
