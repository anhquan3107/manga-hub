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

- `mangahub auth register --username <username> --email <email>`
- `mangahub auth login --username <username>`
- `mangahub auth login --email <email>`
- `mangahub auth logout`
- `mangahub auth status`
- `mangahub auth change-password`

#### Manga Management

- `mangahub manga search <query>`
- `mangahub manga list`
- `mangahub manga info <manga_id>`
- `mangahub manga import --source <mangadex|jikan> --limit <n> --seed-file <path>`

#### Library Operations

- `mangahub library add --manga-id <manga_id> --status <reading|completed|plan-to-read|on-hold|dropped>`
- `mangahub library list`
- `mangahub library update --manga-id <manga_id> --status <status> [--rating <1-10>]`
- `mangahub library remove --manga-id <manga_id>`

#### Progress Synchronization

- `mangahub progress update --manga-id <manga_id> --chapter <number>`
- `mangahub progress update --manga-id <manga_id> --chapter <number> --volume <number>`
- `mangahub progress history --manga-id <manga_id>`
- `mangahub sync connect`
- `mangahub sync connect --user-id <user_id>`
- `mangahub sync disconnect --user-id <user_id>`
- `mangahub sync status`
- `mangahub sync monitor`

#### Notifications

- `mangahub notify subscribe --addr <udp_addr> --client <client_id>`
- `mangahub notify unsubscribe --addr <udp_addr> --client <client_id>`
- `mangahub notify test`

#### Reviews

- `mangahub review add --manga-id <manga_id> --rating <1-10> --text <text>`
- `mangahub review list --manga-id <manga_id> [--limit <n>] [--sort recent|helpful]`
- `mangahub review mine --manga-id <manga_id>`
- `mangahub review helpful --manga-id <manga_id> --user-id <user_id>`

#### gRPC

- `mangahub grpc manga get --id <manga_id>`
- `mangahub grpc manga search --query <text> [--limit <n>]`
- `mangahub grpc progress update --manga-id <manga_id> --chapter <number>`
- `mangahub grpc progress update --manga-id <manga_id> --chapter <number> --volume <number> [--user-id <user_id>] [--force] [--notes <text>]`
- `mangahub grpc user get --user-id <user_id>`
- `mangahub grpc user get --username <username>`
- `mangahub grpc user library --user-id <user_id>`

#### Server Management

- `mangahub server start`
- `mangahub server health`
- `mangahub server status`

#### Chat

- `mangahub chat join`
- `mangahub chat join --manga-id <manga_id>`
- `mangahub chat send "<message>"`
- `mangahub chat send "<message>" --manga-id <manga_id>`
- `mangahub chat history`
- `mangahub chat history --manga-id <manga_id> --limit <n>`

#### Interactive Chat Commands

- `/help` - show chat commands
- `/users` - list online users
- `/quit` - leave chat
- `/pm <user> <message>` - send a private message
- `/manga <manga_id>` - switch rooms
- `/history` - show recent history
- `/status` - show connection status

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
4. Install dependencies if needed:

```bash
go mod tidy
```

5. Create the output and data directories:

```bash
mkdir -p bin data
```

#### Build the local binaries

```bash
go build -o bin/api-server ./cmd/api-server
go build -o bin/tcp-server ./cmd/tcp-server
go build -o bin/udp-server ./cmd/udp-server
go build -o bin/grpc-server ./cmd/grpc-server
go build -o bin/mangahub ./cmd/cli/app
```

#### Start the app locally after build

Run each server in a separate terminal:

```bash
./bin/api-server
./bin/tcp-server
./bin/udp-server
./bin/grpc-server
```

If you want to use the CLI locally:

```bash
./bin/mangahub
```

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

### Test

```bash
GOFLAGS=-tags=sqlite_fts5 go test -v ./...
```

Use the same command with `-race` when you want concurrency checking.

### Docker Compose

Use Docker Compose when you want to run the full MangaHub stack with one command. The Compose files start Redis plus the HTTP, TCP, UDP, and gRPC services, and they load settings from `.env`.

#### Quick start

```bash
cp .env.example .env
docker compose up --build
```

#### 1. Prepare the environment file

Copy the example file and adjust it if needed:

```bash
cp .env.example .env
```

Important values:

- `HTTP_ADDR=:8080`
- `TCP_ADDR=:9090`
- `UDP_ADDR=:9091`
- `GRPC_ADDR=:9092`
- `TCP_SERVER_ADDR=:9093`
- `DB_PATH=./data/mangahub.db`
- `SEED_FILE=./data/manga.sample.json`
- `REDIS_ADDR=redis:6379`

#### 2. Start the development stack

```bash
docker compose up --build
```

After the containers are up, the app is already running. Use these local URLs:

- API: `http://localhost:8080`
- Swagger: `http://localhost:8080/swagger/index.html`
- TCP sync: `localhost:9090`
- UDP notifications: `localhost:9091`
- gRPC: `localhost:9092`

If you want to use the CLI against the Docker stack, build the CLI first and then run the binary in another terminal:

```bash
go build -o bin/mangahub ./cmd/cli/app
./bin/mangahub
```

Development compose starts these containers:

- `redis` - Redis cache at `redis:6379`
- `http` - HTTP API on `http://localhost:8080`
- `tcp` - TCP sync server on `localhost:9090`
- `udp` - UDP notification server on `localhost:9091/udp`
- `grpc` - gRPC server on `localhost:9092`

Development uses the local `./data` directory as a bind mount, so SQLite data is saved on your machine.

#### 3. Start the production stack

```bash
docker compose -f docker-compose.prod.yml up --build
```

Production compose uses the published image and named volumes:

- `mangahub-data` for SQLite files
- `redis-data` for Redis persistence

#### 4. Useful Docker commands

```bash
docker compose ps
docker compose logs -f
docker compose down
```

#### 5. Docker behavior notes

- The API, TCP, UDP, and gRPC services all depend on Redis being available.
- The API server reads the same `.env` file in both local and containerized setups.
- If you change `SEED_FILE` or `DB_PATH`, update the mounted paths in Compose accordingly.
- To rebuild after code changes, run `docker compose up --build` again.

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
