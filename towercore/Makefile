.PHONY: build run test fmt docker-up docker-down

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

docker-up:
	docker compose -f docker-compose.yml up -d --build

docker-down:
	docker compose -f docker-compose.yml down
