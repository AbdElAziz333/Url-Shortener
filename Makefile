ifneq (,$(wildcard .env))
	include .env
	export
endif

# Protobuf

generate_proto:
	cd shortener && mkdir -p internal/pb && protoc --proto_path=../proto --go_out=internal/pb --go_opt=paths=source_relative --go-grpc_out=internal/pb --go-grpc_opt=paths=source_relative ../proto/shortener.proto
	cd redirect && mkdir -p internal/pb && protoc --proto_path=../proto --go_out=internal/pb --go_opt=paths=source_relative --go-grpc_out=internal/pb --go-grpc_opt=paths=source_relative ../proto/redirect.proto
	cd gateway && mkdir -p internal/pb/shortener internal/pb/redirect && protoc --proto_path=../proto --go_out=internal/pb/shortener --go_opt=paths=source_relative --go_opt=Mshortener.proto=aziz.dev/gateway/internal/pb/shortener --go-grpc_out=internal/pb/shortener --go-grpc_opt=paths=source_relative --go-grpc_opt=Mshortener.proto=aziz.dev/gateway/internal/pb/shortener ../proto/shortener.proto
	cd gateway && protoc --proto_path=../proto --go_out=internal/pb/redirect --go_opt=paths=source_relative --go_opt=Mredirect.proto=aziz.dev/gateway/internal/pb/redirect --go-grpc_out=internal/pb/redirect --go-grpc_opt=paths=source_relative --go-grpc_opt=Mredirect.proto=aziz.dev/gateway/internal/pb/redirect ../proto/redirect.proto

# Run Locally

run_gateway:
	cd gateway && go run cmd/main.go

run_shortener:
	cd shortener && go run cmd/main.go

run_redirect:
	cd redirect && go run cmd/main.go

# Run Unit Tests

run_gateway_unit_tests:
	go test ./gateway/...
run_shortener_unit_tests:
	go test ./shortener/...
run_redirect_unit_tests:
	go test ./redirect/...
run_all_unit_tests:
	run_gateway_unit_tests
	run_shortener_unit_tests
	run_redirect_unit_tests

# Run Integration Tests

run_gateway_integration_tests:
	go test -tag=integration ./gateway/...
run_shortener_integration_tests:
	go test -tag=integration ./shortener/...
run_redirect_integration_tests:
	go test -tag=integration ./redirect/...
run_all_integration_tests:
	run_gateway_integration_tests
	run_shortener_integration_tests
	run_redirect_integration_tests

# SQL Migrations

migrate_gateway:
	migrate -database postgresql://aziz:aziz333@localhost:5432/gateway?sslmode=disable -path gateway/migrations up

migrate_shortener:
	migrate -database postgresql://aziz:aziz333@localhost:5432/shortener?sslmode=disable -path shortener/migrations up

run_all_migrations:
	migrate_gateway
	migrate_shortener

# Docker

docker_build_gateway:
	cd gateway && docker build -t abdelaziz333/gateway:${APP_VERSION} .

docker_build_shortener:
	cd shortener && docker build -t abdelaziz333/shortener:${APP_VERSION} .

docker_build_redirect:
	cd redirect && docker build -t abdelaziz333/redirect:${APP_VERSION} .

docker_build_all:
	docker compose build gateway shortener redirect

build_each_separately: docker_build_gateway docker_build_shortener docker_build_redirect

docker_push_gateway: docker_build_gateway
	docker push abdelaziz333/gateway:${APP_VERSION}

docker_push_shortener: docker_build_shortener
	docker push abdelaziz333/shortener:${APP_VERSION}

docker_push_redirect: docker_build_redirect
	docker push abdelaziz333/redirect:${APP_VERSION}

docker_push_all: docker_push_gateway docker_push_shortener docker_push_redirect

docker_run_apps:
	docker compose --profile app up

docker_up:
	docker compose --profile all up -d

docker_down:
	docker compose --profile all down
