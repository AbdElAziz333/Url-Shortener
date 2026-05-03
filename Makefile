# run locally

run_gateway:
	cd gateway && go run cmd/main.go

run_shortener:
	cd shortener && go run cmd/main.go

run_redirect:
	cd redirect && go run cmd/main.go

run_analytics:
	cd analytics && go run cmd/main.go

# run docker
# run kubernetes

# go commands
# migrate commands

migrate_gateway:
	migrate -database postgresql://aziz:aziz333@localhost:5432/gateway?sslmode=disable -path gateway/migrations up

migrate_shortener:
	migrate -database postgresql://aziz:aziz333@localhost:5432/shortener?sslmode=disable -path shortener/migrations up

migrate_analytics:
	migrate -database postgresql://aziz:aziz333@localhost:5432/analytics?sslmode=disable -path analytics/migrations up
# docker commands
