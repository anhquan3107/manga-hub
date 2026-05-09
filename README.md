# MangaHub

## API Documentation

MangaHub is a Go-based manga tracking system created for the IT096IU Network Programming term project. It demonstrates network application development through the five required communication protocols in one application:

- HTTP for authentication, manga browsing, library management, reviews, and health checks
- TCP for reading progress synchronization
- UDP for notifications
- WebSocket for realtime chat and room management
- gRPC for internal service-to-service communication

The implemented system includes SQLite persistence, Swagger API documentation, Redis caching for frequently accessed data, Docker Compose deployment, and a CLI for terminal-based interaction.

### Main HTTP API

#### System

- `GET /health`
- `GET /swagger/*any`

#### Authentication

- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/logout`
- `POST /auth/change-password`

#### Manga

- `GET /manga`
- `GET /manga/:id`
- `POST /manga`
- `PUT /manga/:id`
- `DELETE /manga/:id`

#### Reviews

- `GET /manga/:id/reviews`
- `POST /manga/:id/reviews`
- `GET /manga/:id/reviews/me`
- `POST /manga/:id/reviews/:user_id/helpful`

#### User Library and Progress

- `GET /users/me`
- `GET /users/library`
- `POST /users/library`
- `PUT /users/library/:id`
- `DELETE /users/library/:id`
- `PUT /users/progress`
- `GET /users/progress/history`
- `POST /users/pm`

#### Realtime Chat

- `GET /ws/chat`
- `GET /rooms/users`
- `GET /rooms/:room/users`
- `GET /rooms/:room/history`

### gRPC Methods

#### Manga Service

- `GetManga`
- `SearchManga`

#### Progress Service

- `UpdateProgress`

### CLI Commands

#### Authentication

- `mangahub auth register`
- `mangahub auth login`
- `mangahub auth logout`
- `mangahub auth status`
- `mangahub auth change-password`

#### Manga Management

- `mangahub manga search`
- `mangahub manga list`
- `mangahub manga info`

#### Library Operations

- `mangahub library add`
- `mangahub library list`
- `mangahub library update`
- `mangahub library remove`

#### Progress Synchronization

- `mangahub progress update`
- `mangahub progress history`
- `mangahub progress sync`
- `mangahub progress sync-status`
- `mangahub sync connect`
- `mangahub sync disconnect`
- `mangahub sync status`
- `mangahub sync monitor`

#### Notifications

- `mangahub notify subscribe`
- `mangahub notify unsubscribe`
- `mangahub notify test`

#### Chat

- `mangahub chat join`
- `mangahub chat send`
- `mangahub chat history`

## Setup Instructions

### Prerequisites

- Go 1.25 or later
- SQLite
- Docker and Docker Compose
- Redis for the optional cache layer and the container stack

### Local Setup

1. Copy the environment template:

```bash
cp .env.example .env
```

2. Update values in `.env` if needed.
3. Make sure the seed data file exists at the configured `SEED_FILE` path.
4. Start the servers or run the project through Docker Compose.

### Environment Variables

The application reads these variables:

- `HTTP_ADDR`
- `TCP_ADDR`
- `UDP_ADDR`
- `GRPC_ADDR`
- `TCP_SERVER_ADDR`
- `DB_PATH`
- `SEED_FILE`
- `JWT_SECRET`
- `ALLOWED_ORIGIN`
- `REDIS_ADDR`
- `REDIS_PASSWORD`

### Build

```bash
go build -o bin/api-server ./cmd/api-server
go build -o bin/tcp-server ./cmd/tcp-server
go build -o bin/udp-server ./cmd/udp-server
go build -o bin/grpc-server ./cmd/grpc-server
go build -o bin/mangahub ./cmd/cli/app
```

### Test

```bash
GOFLAGS=-tags=sqlite_fts5 go test -v ./...
```

Use the same command with `-race` when you want concurrency checking.

### Docker Compose

Development:

```bash
docker compose up --build
```

Production:

```bash
docker compose -f docker-compose.prod.yml up --build
```

Both compose files include the Redis service and the protocol servers.

## Architecture Overview

MangaHub is organized as a multi-service Go application with a shared SQLite data layer.

### Service Layout

- `cmd/api-server` - HTTP API entrypoint
- `cmd/tcp-server` - TCP progress sync server
- `cmd/udp-server` - UDP notification server
- `cmd/grpc-server` - gRPC service entrypoint
- `cmd/cli/app` - CLI entrypoint

### Internal Packages

- `internal/api` - router, handlers, middleware
- `internal/auth` - authentication service and middleware
- `internal/manga` - manga business logic and caching
- `internal/user` - user, library, and progress logic and caching
- `internal/review` - review and rating logic and caching
- `internal/tcp` - TCP synchronization server
- `internal/udp` - UDP notification server
- `internal/websocket` - WebSocket chat hub and handler
- `internal/grpc` - gRPC service implementation
- `internal/cache` - Redis client wrapper
- `pkg/database` - SQLite store and repositories
- `pkg/models` - shared models
- `proto` - protobuf definitions and generated code

### Request Flow

1. The HTTP API handles authentication, manga browsing, library updates, reviews, and health checks.
2. The TCP server broadcasts reading progress updates to connected clients.
3. The UDP server manages notification registration and broadcasts.
4. The WebSocket server handles realtime chat rooms and private messaging.
5. The gRPC server exposes internal manga and progress methods.
6. Redis is used as an optional cache layer for frequently accessed read paths in manga, review, and user services.

### Data Storage

The application stores data in SQLite with tables for users, manga, user progress, reviews, chat messages, and private messages. Seed manga data is loaded from `data/manga.sample.json` at startup.
