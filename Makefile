DB_URL=postgresql://postgres:postgres@localhost:5432/ricesafe?sslmode=disable
DB_URL_DOCKER=postgresql://postgres:postgres@db:5432/ricesafe?sslmode=disable
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

.PHONY: docker-up docker-down migrate-up migrate-down create-migration
