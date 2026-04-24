# MangaHub

Go-based manga platform with HTTP, TCP, UDP, WebSocket, and gRPC services.

## Week 1 (Foundation) Status

Completed:
- Go project structure with separate commands under [cmd](cmd)
- Gin HTTP API server in [cmd/api-server/main.go](cmd/api-server/main.go)
- Register/Login endpoints in [internal/api/router.go](internal/api/router.go)
- JWT auth service and middleware in [internal/auth/service.go](internal/auth/service.go) and [internal/auth/middleware.go](internal/auth/middleware.go)
- SQLite schema and data access in [pkg/database/sqlite.go](pkg/database/sqlite.go)

## Data Collection Outputs

- Manual source dataset: [data/manga.manual.json](data/manga.manual.json)
- Merged dataset used by API seed: [data/manga.sample.json](data/manga.sample.json)
- Collection report: [data/collection_report.json](data/collection_report.json)

Run collector:

```bash
go run ./cmd/data-collector
```

## System Architecture Components

### 1) HTTP REST API

Start:

```bash
go run ./cmd/api-server
```

Key endpoints:
- POST `/auth/register`
- POST `/auth/login`
- GET `/manga`
- GET `/manga/:id`
- POST `/users/library`
- GET `/users/library`
- PUT `/users/progress`
- GET `/ws/chat`

`GET /users/library` returns both:
- `items`: full list
- `reading_lists`: grouped by `reading`, `completed`, and `plan_to_read`

### 2) TCP Progress Sync

Start:

```bash
go run ./cmd/tcp-server
```

Server implementation: [internal/tcp/server.go](internal/tcp/server.go)

### 3) UDP Notification Broadcast

Start:

```bash
go run ./cmd/udp-server
```

Server implementation: [internal/udp/server.go](internal/udp/server.go)

### 4) WebSocket Chat

WebSocket endpoint is served by API server at `/ws/chat?token=<jwt>`.

Hub and handler:
- [internal/websocket/hub.go](internal/websocket/hub.go)
- [internal/websocket/handler.go](internal/websocket/handler.go)

### 5) gRPC Internal Service

Protocol definition: [proto/mangahub.proto](proto/mangahub.proto)

Start gRPC server:

```bash
go run ./cmd/grpc-server
```

Simple gRPC client integration:

```bash
go run ./cmd/grpc-client -method search -query one -limit 5
go run ./cmd/grpc-client -method get -id one-piece
go run ./cmd/grpc-client -method progress -user user-1 -manga one-piece -chapter 10 -status reading
```

### 6) Database Layer

Database layer and schema:
- [pkg/database/sqlite.go](pkg/database/sqlite.go)

Default environment values are in [.env.example](.env.example).

## Environment

Copy values from [.env.example](.env.example) into your environment before running services.

