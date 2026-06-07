.PHONY: build run migrate test setup clean docker-build docker-up docker-down

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

test:
	go test -v ./...

setup:
	go run ./cmd/migrate -all

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