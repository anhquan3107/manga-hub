# MangaHub

MangaHub is a Go-based manga platform with a CLI for authentication, manga browsing, library tracking, progress sync, chat, and notifications.

## Quick Start

### 1. Build the CLI

```bash
go build -o build/mangahub ./cmd/cli/app
```

### 2. Start the backend services

The CLI talks to the API server and the realtime services. Start the server from the build directory:

```bash
cd build
./mangahub server start
```

## CLI Overview

```bash
mangahub auth <register|login|logout|status|change-password>
mangahub manga <search|list|info>
mangahub library <add|list|remove|update>
mangahub progress <update|history|sync|sync-status>
mangahub chat <join|send|history>
mangahub sync <connect|disconnect|status|monitor>
mangahub notify <subscribe|unsubscribe|preferences|test>
```

## Authentication

Auth tokens are saved per terminal session under `~/.mangahub/<session_id>/token`, so two terminals can stay logged in as different users at the same time.

### Register

```bash
mangahub auth register --username <username> --email <email>
```

The CLI prompts for password and confirmation.

### Login

```bash
mangahub auth login --username <username>
```

The CLI prompts for the password and stores the returned token for the current terminal session.

### Status

```bash
mangahub auth status
```

Shows the active session, token status, and current user information.

### Logout

```bash
mangahub auth logout
```

Clears the token for the current terminal session only.

## Manga

```bash
mangahub manga search "one piece"
mangahub manga list
mangahub manga info <manga_id>
```

## Library

Requires login.

```bash
mangahub library add --manga-id <manga_id> --status reading
mangahub library list
mangahub library remove <manga_id>
mangahub library update --manga-id <manga_id> --status completed --rating 9
```

## Progress

Requires login.

```bash
mangahub progress update --manga-id <manga_id> --chapter 12
mangahub progress history --manga-id <manga_id>
mangahub progress sync --user-id <user_id>
mangahub progress sync-status
```

The progress update command also broadcasts updates to the TCP sync service.

## Chat

Join a realtime room over WebSocket:

```bash
mangahub chat join
```

Inside chat:
- Type any message and press Enter to send
- Use `/users` to list online users
- Use `/history` to show recent messages
- Use `/manga <manga_id>` to switch rooms
- Use `/pm <username> <message>` to send a private message
- Use `/quit` to leave chat

Examples:

```bash
mangahub chat send "hello everyone"
mangahub chat history --limit 20
```

Private messages are stored in the database and pushed in realtime to connected recipients.

## Sync

TCP sync is available for progress coordination.

```bash
mangahub sync connect --user-id <user_id>
mangahub sync disconnect --user-id <user_id>
mangahub sync status
mangahub sync monitor
```

## Notifications

UDP notifications are handled through the `notify` command.

```bash
mangahub notify subscribe --client-id <client_id>
mangahub notify unsubscribe --client-id <client_id>
mangahub notify preferences --client-id <client_id>
mangahub notify test --manga-id <manga_id>
```

## Project Structure

- `cmd/api-server` - HTTP API entrypoint
- `cmd/cli` - CLI entrypoint and commands
- `internal/api` - HTTP handlers and routing
- `internal/auth` - JWT auth service and middleware
- `internal/chat` - chat business logic
- `internal/websocket` - realtime chat hub and handler
- `internal/user` - user and library services
- `pkg/database` - SQLite repositories and schema setup
- `pkg/models` - request and response models

## Notes

- The API server uses SQLite by default at `./data/mangahub.db`
- Seed manga data comes from `./data/manga.sample.json`
- The CLI is designed to work with the current session-scoped token storage so multiple terminals can stay authenticated independently


