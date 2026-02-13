# Docker

## Quick Start

```bash
docker build -t portblock .
docker run -v $(pwd)/api.yaml:/spec.yaml -p 4000:4000 portblock serve /spec.yaml --port 4000
```

## Docker Compose

```bash
docker compose up
```

The included `docker-compose.yml` mounts `examples/` and serves the todo API.

## Running Tests in Docker

```bash
docker run -v $(pwd):/workspace portblock test /workspace/api.yaml /workspace/tests.yaml
```

## Multi-Stage Build

The Dockerfile uses a multi-stage build:
1. **Build stage**: compiles the Go binary
2. **Runtime stage**: Alpine-based, ~15MB final image

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o portblock .

FROM alpine:3.19
COPY --from=builder /app/portblock /usr/local/bin/
ENTRYPOINT ["portblock"]
```
