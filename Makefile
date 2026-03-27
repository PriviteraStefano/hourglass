.PHONY: build run migrate test clean docker-build docker-up docker-down

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

test:
	go test -v ./...

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