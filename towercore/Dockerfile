FROM golang:1.22-alpine AS builder
WORKDIR /app

COPY go.mod ./
COPY go.sum ./
COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations
COPY pkg ./pkg

RUN go build -o towercore ./cmd/api

FROM alpine:3.20
WORKDIR /app

COPY --from=builder /app/towercore /app/towercore

EXPOSE 8000
ENTRYPOINT ["/app/towercore"]
