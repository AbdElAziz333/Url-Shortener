# Run Locally

run_gateway:
	cd gateway && go run cmd/main.go

run_shortener:
	cd shortener && go run cmd/main.go

run_redirect:
	cd redirect && go run cmd/main.go

run_analytics:
	cd analytics && go run cmd/main.go

# SQL Migrations

migrate_gateway:
	migrate -database postgresql://aziz:aziz333@localhost:5432/gateway?sslmode=disable -path gateway/migrations up

migrate_shortener:
	migrate -database postgresql://aziz:aziz333@localhost:5432/shortener?sslmode=disable -path shortener/migrations up

migrate_analytics:
	migrate -database postgresql://aziz:aziz333@localhost:5432/analytics?sslmode=disable -path analytics/migrations up

# Docker

docker_build_gateway:
	cd gateway && docker build -t abdelaziz333/gateway:${APP_VERSION} .

docker_build_shortener:
	cd shortener && docker build -t abdelaziz333/shortener:${APP_VERSION} .

docker_build_redirect:
	cd redirect && docker build -t abdelaziz333/redirect:${APP_VERSION} .

docker_build_analytics:
	cd analytics && docker build -t abdelaziz333/analytics:${APP_VERSION} .

docker_build_all:
	docker compose build gateway shortener redirect analytics

build_each_separately: docker_build_gateway docker_build_shortener docker_build_redirect docker_build_analytics

docker_up:
	docker compose up -d

docker_down:
	docker compose down

docker_up_infra:
	docker compose -f docker-compose.infra.yaml up -d

docker_infra_down:
	docker compose -f docker-compose.infra.yaml down -d