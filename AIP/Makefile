# AIP — Antosc Intelligence Platform
# Uso: make <target>

APP       := aip
BIN_DIR   := bin
GO        := go
MODS      := health_score

.PHONY: all build clean tidy vet smoke migrate seed retrain drift embeddings report-weekly report-monthly predict run-api run-scheduler help

help:
	@echo "Targets:"
	@echo "  make build            - compila todos os binários para ./bin"
	@echo "  make tidy             - go mod tidy"
	@echo "  make vet              - go vet ./..."
	@echo "  make migrate          - aplica migrations (precisa AIP_DB_* ou DATABASE_URL)"
	@echo "  make run-api          - arranca a API"
	@echo "  make run-scheduler    - arranca o scheduler de ingestão"
	@echo "  make predict          - previsões em lote (todas as torres)"
	@echo "  make report-weekly    - envia relatórios semanais por persona"
	@echo "  make report-monthly   - envia relatórios mensais por persona"
	@echo "  make smoke            - smoke test HTTP (API tem de estar a correr)"
	@echo "  make seed             - cria admin + users por persona (idempotente)"
	@echo "  make retrain          - re-treino modelos Python (health + anomaly)"
	@echo "  make drift            - verifica drift (ANOMALY_TRAIN_CSV=...)"
	@echo "  make embeddings       - PCA cross-tower (ANOMALY_TRAIN_CSV=...)"
	@echo "  make clean            - remove ./bin"

all: build

tidy:
	$(GO) mod tidy

vet:
	$(GO) vet ./...

build: tidy
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/api ./cmd/api
	$(GO) build -o $(BIN_DIR)/scheduler ./cmd/scheduler
	$(GO) build -o $(BIN_DIR)/report_job ./cmd/report_job
	$(GO) build -o $(BIN_DIR)/predict_batch ./cmd/predict_batch
	$(GO) build -o $(BIN_DIR)/weekly_report ./cmd/weekly_report
	$(GO) build -o $(BIN_DIR)/seed_users ./cmd/seed_users
	@echo "binários em $(BIN_DIR)/"

clean:
	rm -rf $(BIN_DIR)

migrate:
	@bash scripts/migrate.sh

run-api: build
	./$(BIN_DIR)/api

run-scheduler: build
	./$(BIN_DIR)/scheduler

predict: build
	PREDICT_MODELS=$(or $(PREDICT_MODELS),health_score) ./$(BIN_DIR)/predict_batch

report-weekly: build
	REPORT_PERIOD=weekly ./$(BIN_DIR)/report_job

report-monthly: build
	REPORT_PERIOD=monthly ./$(BIN_DIR)/report_job

smoke:
	@bash scripts/smoke_test.sh

seed: build
	./$(BIN_DIR)/seed_users

retrain:
	@bash scripts/retrain_models.sh

drift:
	@bash scripts/check_drift.sh $${ANOMALY_TRAIN_CSV}

embeddings:
	@test -n "$${ANOMALY_TRAIN_CSV}" || (echo "define ANOMALY_TRAIN_CSV=features_wide.csv" && exit 1)
	cd python && PYTHONPATH=. python3 -m training.build_embeddings $${ANOMALY_TRAIN_CSV}
