# Implementation Plan - gRPC Integration for Microservices

This plan describes how to integrate gRPC for internal service-to-service communication between the `gateway`, `shortener`, and `redirect` microservices.

## User Review Required

> [!IMPORTANT]
> - We will run the gRPC servers on new ports (`50051` for `shortener`, `50052` for `redirect`) alongside the existing HTTP servers. This preserves the existing HTTP health check endpoints and ensures that all current HTTP unit tests continue to pass without changes.
> - The gateway will connect to these microservices using gRPC instead of HTTP reverse-proxying.
> - We will implement a custom gRPC client circuit breaker interceptor using `gobreaker` in the gateway to handle failures on internal calls.

## Proposed Changes

### 1. Protobuf Definitions

We will create a `proto/` directory at the root and define:
- [NEW] `proto/shortener.proto`
- [NEW] `proto/redirect.proto`

#### `proto/shortener.proto`
Defines the `ShortenerService` with the following RPCs:
- `GetAllLinks` (to retrieve all links for a user)
- `CreateLink` (to create a shortened link)
- `UpdateExpiry` (to update link expiration)
- `DeleteLink` (to deactivate/delete a link)

#### `proto/redirect.proto`
Defines the `RedirectService` with the following RPCs:
- `ResolveCode` (to resolve a shortened code to its original URL)

### 2. Shortener Service Changes

We will generate the protobuf Go code under `shortener/internal/pb/` and implement the gRPC server.

#### [NEW] `shortener/internal/pb/`
Contains the generated gRPC files:
- `shortener.pb.go`
- `shortener_grpc.pb.go`

#### [NEW] [server.go](file:///home/aziz/Desktop/Programming/Projects/url-shortener/shortener/internal/server/grpc.go)
Defines the `ShortenerServiceServer` implementation which delegates calls to the core `link.Service`.

#### [MODIFY] [main.go](file:///home/aziz/Desktop/Programming/Projects/url-shortener/shortener/cmd/main.go)
Start the gRPC server in a separate goroutine on port `50051` (or via env `GRPC_PORT`) alongside the existing Gin HTTP server.

### 3. Redirect Service Changes

We will generate the protobuf Go code under `redirect/internal/pb/` and implement the gRPC server.

#### [NEW] `redirect/internal/pb/`
Contains the generated gRPC files:
- `redirect.pb.go`
- `redirect_grpc.pb.go`

#### [NEW] [server.go](file:///home/aziz/Desktop/Programming/Projects/url-shortener/redirect/internal/server/grpc.go)
Defines the `RedirectServiceServer` implementation which delegates calls to the core `resolve.Service`.

#### [MODIFY] [main.go](file:///home/aziz/Desktop/Programming/Projects/url-shortener/redirect/cmd/main.go)
Start the gRPC server in a separate goroutine on port `50052` (or via env `GRPC_PORT`) alongside the existing Gin HTTP server.

### 4. Gateway Service Changes

We will generate the protobuf Go code under `gateway/internal/pb/shortener/` and `gateway/internal/pb/redirect/` and implement the client logic.

#### [NEW] `gateway/internal/pb/shortener/` and `gateway/internal/pb/redirect/`
Contains the generated gRPC client files.

#### [NEW] [client.go](file:///home/aziz/Desktop/Programming/Projects/url-shortener/gateway/internal/grpcclient/client.go)
Initializes the gRPC client connections for both services, wrapped in a gRPC client interceptor that integrates the `gobreaker` circuit breaker.

#### [NEW] [handler.go](file:///home/aziz/Desktop/Programming/Projects/url-shortener/gateway/internal/grpcclient/handler.go)
Provides HTTP handler functions that call the downstream gRPC clients and map gRPC status codes/errors to standard HTTP response formats.

#### [MODIFY] [main.go](file:///home/aziz/Desktop/Programming/Projects/url-shortener/gateway/cmd/main.go)
Initialize the gRPC clients and replace the Gin route definitions using `proxy.ReverseProxy` with our new handlers.

### 5. Infrastructure Changes

#### [MODIFY] [docker-compose.yaml](file:///home/aziz/Desktop/Programming/Projects/url-shortener/docker-compose.yaml)
Update the environment variables for service URLs to point to the gRPC servers:
- `SHORTENER_SERVICE_URL: shortener:50051`
- `REDIRECT_SERVICE_URL: redirect:50052`
Also expose the gRPC ports in the docker compose services if needed.

## Verification Plan

### Automated Tests
- Run existing unit tests: `cd gateway && go test ./...`, etc.
- Write new unit/integration tests for the gRPC servers and the gateway gRPC client wrapper handlers.

### Manual Verification
- Start all services using `docker compose --profile all up -d`
- Run integration tests/smoke tests by making HTTP calls to the gateway (e.g. creating a link, resolving it) and ensuring internal gRPC communication works.
