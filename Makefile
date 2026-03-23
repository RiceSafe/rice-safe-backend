-include .env
export

DB_URL=postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:5432/${POSTGRES_DB}?sslmode=disable
DB_URL_DOCKER=postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@db:5432/${POSTGRES_DB}?sslmode=disable
NETWORK=rice-safe-backend_rice-safe-net

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

migrate-up:
	docker run --rm -v $(PWD)/migrations:/migrations --network $(NETWORK) migrate/migrate -path=/migrations/ -database "$(DB_URL_DOCKER)" up

migrate-down:
	docker run --rm -v $(PWD)/migrations:/migrations --network $(NETWORK) migrate/migrate -path=/migrations/ -database "$(DB_URL_DOCKER)" down

create-migration:
	docker run --rm -v $(PWD)/migrations:/migrations migrate/migrate create -ext sql -dir /migrations -seq $(name)

test:
	go test -v ./tests/...

dev:
	air

swagger:
	swag init -g cmd/api/main.go --parseDependency --parseInternal
	@echo "Swagger docs generated successfully!"

.PHONY: docker-up docker-down migrate-up migrate-down create-migration test dev swagger
