# Docker для Go

## Multi-Stage Build (production)

```dockerfile
# Stage 1: Build
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Кэшируем зависимости (слой меняется редко)
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники и собираем
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /app/server ./cmd/server

# Stage 2: Runtime
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /app/server /server
COPY --from=builder /app/migrations /migrations

USER nonroot:nonroot
EXPOSE 8080 50051

ENTRYPOINT ["/server"]
```

### Размеры образов

```
golang:1.24          → ~800 MB (с toolchain)
golang:1.24-alpine   → ~250 MB
alpine:3.19          → ~7 MB (+ бинарник)
distroless/static    → ~2 MB (+ бинарник)
scratch              → 0 MB (+ бинарник)

Типичный Go бинарник: 10-30 MB
Финальный образ: 12-35 MB (distroless + бинарник)
```

### Build flags

```bash
# CGO_ENABLED=0 — статическая линковка (не зависит от libc)
# -ldflags="-s -w" — убрать debug info (~30% меньше бинарник)
# -s: убрать symbol table
# -w: убрать DWARF debug info

# Version injection
go build -ldflags="-s -w \
  -X main.version=$(git describe --tags) \
  -X main.commit=$(git rev-parse --short HEAD) \
  -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o server ./cmd/server
```

## .dockerignore

```
.git
.github
.vscode
*.md
docs/
tmp/
vendor/  # если используешь go mod download
```

## Docker Compose (development)

```yaml
services:
  app:
    build:
      context: .
      dockerfile: Dockerfile
      target: builder  # останавливаемся на стадии builder для dev
    command: go run ./cmd/server
    volumes:
      - .:/app
    ports:
      - "8080:8080"
      - "50051:50051"
    environment:
      - DATABASE_URL=postgres://user:pass@postgres:5432/mydb?sslmode=disable
      - REDIS_URL=redis:6379
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_started

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
      POSTGRES_DB: mydb
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U user"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  kafka:
    image: confluentinc/confluent-local:7.5.0
    ports:
      - "9092:9092"

volumes:
  pgdata:
```

## Best Practices

```
1. Multi-stage builds — маленький финальный образ
2. nonroot user — безопасность (не root!)
3. distroless или scratch — минимальная attack surface
4. CGO_ENABLED=0 — статический бинарник
5. Кэширование go mod download — быстрые ребилды
6. .dockerignore — не копировать лишнее
7. Health check в Dockerfile:
   HEALTHCHECK --interval=30s --timeout=3s \
     CMD ["/server", "healthcheck"] || exit 1
8. Не хранить secrets в image — через env vars / volumes
```
