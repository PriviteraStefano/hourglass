.PHONY: build run migrate test setup clean docker-build docker-up docker-down seed seed-demo demo-up demo-migrate demo-seed demo-recover-db demo-redeploy

BINARY_NAME=hourglass
MIGRATIONS_DIR=migrations

build:
	go build -o bin/$(BINARY_NAME) ./cmd/server

run:
	go run ./cmd/server

migrate-up:
	go run ./cmd/migrate -up -dir $(MIGRATIONS_DIR)

migrate-down:
	go run ./cmd/migrate -down -dir $(MIGRATIONS_DIR)

migrate-all:
	go run ./cmd/migrate -all -dir $(MIGRATIONS_DIR)

# Apply the demo seed after all migrations (post-011/012 schema). Idempotent.
#
# `seed`      → the local dev DB inside the docker container (no host psql needed)
# `seed-demo` → any Postgres over a connection URL (host `psql` required); use for
#               the deployed demo environment
#
#   make seed-demo DATABASE_URL_DEMO="postgres://user:pass@demo-host:5432/hourglass?sslmode=require"
DB_CONTAINER ?= hourglass-postgres
DB_NAME      ?= hourglass
DB_USER      ?= hourglass
seed:
	docker cp scripts/seed_demo.sql $(DB_CONTAINER):/tmp/seed_demo.sql
	docker exec $(DB_CONTAINER) psql -U $(DB_USER) -d $(DB_NAME) -v ON_ERROR_STOP=1 -f /tmp/seed_demo.sql

DATABASE_URL_DEMO ?=
seed-demo:
	@if [ -z "$(DATABASE_URL_DEMO)" ]; then \
		echo "ERROR: set DATABASE_URL_DEMO, e.g."; \
		echo '  make seed-demo DATABASE_URL_DEMO="postgres://user:pass@demo-host:5432/hourglass?sslmode=require"'; \
		exit 1; \
	fi
	psql "$(DATABASE_URL_DEMO)" -v ON_ERROR_STOP=1 -f scripts/seed_demo.sql

test:
	go test -v ./...

setup:
	go run ./cmd/migrate -all
	$(MAKE) seed

clean:
	rm -rf bin/

docker-build:
	docker build -t hourglass:latest .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

db-init:
	docker exec -i hourglass-postgres psql -U hourglass -d hourglass < $(MIGRATIONS_DIR)/001_init.up.sql

# ── Demo environment (deploy/demo) ─────────────────────────────────────────────
# See hourglass-vault ADR-BE-015 and openwiki/operations/demo-deployment.md.
DEMO_DIR ?= deploy/demo
demo-up:
	docker compose -f $(DEMO_DIR)/docker-compose.yml up -d --build
demo-migrate:
	docker compose -f $(DEMO_DIR)/docker-compose.yml run --rm migrate
demo-seed:
	docker compose -f $(DEMO_DIR)/docker-compose.yml run --rm seed
# Realign postgres password with .env after a pgdata volume exists (28P01 recovery).
demo-recover-db:
	docker compose -f $(DEMO_DIR)/docker-compose.yml run --rm recover-db
demo-redeploy:
	git pull && docker compose -f $(DEMO_DIR)/docker-compose.yml up -d --build && docker image prune -f