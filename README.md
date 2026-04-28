# MangaHub

MangaHub is a platform and CLI tool to discover, track, and discuss Manga.

## Getting Started

### Starting the Server

The main API server must be running for the CLI to work.

```bash
# Go to your project root
go run cmd/api-server/main.go
```

### Installing the CLI

**Option 1: Quick Local Use (Recommended)**
Build the executable into the build folder and run it directly:

```bash
# Build the executable into the build folder
go build -o build/mangahub cmd/cli/main.go

# Navigate to build folder
cd build

# Run with ./ prefix (tells shell to look in current directory)
./mangahub auth login --username <your_user> --password <your_password>
```

*Alternatively, if you want to use just `mangahub` without the `./` prefix in the build folder:*
```bash
# Set up an alias for just that session
alias mangahub="./mangahub"

# Now you can use it directly:
mangahub auth login --username <your_user> --password <your_password>
```

**Option 2: Global Installation**
Install it globally so you can use `mangahub` from anywhere:

```bash
# Build the executable into the build folder
go build -o build/mangahub cmd/cli/main.go

# Move it to your system path (Linux/macOS)
sudo mv build/mangahub /usr/local/bin/

# Now you can use it from anywhere:
mangahub auth login --username <your_user> --password <your_password>
```

## CLI Usage

The CLI commands are mapped directly to the active endpoints, designed with clean architecture and stored in `cmd/cli/commands/`.

### 1. Authentication

Authentication generates a token securely saved to `~/.mangahub/token` for subsequent requests.

**Register:**
```bash
mangahub auth register --username <your_user> --email <your_email> --password <your_password>
```

**Login:**
```bash
mangahub auth login --username <your_user> --password <your_password>
```

### 2. Manga Management

**Search and List:**
```bash
mangahub manga list
mangahub manga search --query "One Piece"
```

**View Details:**
```bash
mangahub manga info <manga_id>
```

### 3. Library & Progress

_Note: Requires you to be logged in first via `auth login`._

**Add Manga to Library:**
```bash
mangahub library add --manga-id <manga_id> --status reading
```

**List your Library:**
```bash
mangahub library list
```

**Update Reading Progress:**
```bash
mangahub progress update --manga-id <manga_id> --chapter <chapter_number>
```

### 4. Interactive Chat

Join real-time discussion across MangaHub via WebSocket.

```bash
mangahub chat join
```
- Type any message and press enter to send.
- Use `/users` to list online users.
- Use `/quit` to exit the chat room.


