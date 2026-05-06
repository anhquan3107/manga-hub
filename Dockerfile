# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Enable SQLite FTS5 for full-text search support
ENV GOFLAGS=-tags=sqlite_fts5

# Install build dependencies
RUN apk add --no-cache gcc musl-dev
# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download


# Copy source code
COPY . .
# Build all binaries
RUN go build -o /app/bin/api-server ./cmd/api-server
RUN go build -o /app/bin/udp-server ./cmd/udp-server
RUN go build -o /app/bin/tcp-server ./cmd/tcp-server
RUN go build -o /app/bin/grpc-server ./cmd/grpc-server
RUN go build -o /app/bin/mangahub ./cmd/cli/app

# Final stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates libc6-compat

# Copy binaries from builder
COPY --from=builder /app/bin/* /usr/local/bin/

# Create data directory
RUN mkdir -p /app/data

# Copy seed data if exists
COPY --from=builder /app/data/manga.sample.json /app/data/manga.sample.json

# The container will run api-server by default
CMD ["api-server"]
