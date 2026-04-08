# CI/CD для Go

## Linting

```yaml
# .golangci.yml
linters:
  enable:
    - errcheck       # проверка необработанных ошибок
    - govet          # go vet
    - staticcheck    # мощный статический анализ
    - gosimple       # упрощение кода
    - ineffassign    # неиспользуемые присваивания
    - unused         # неиспользуемый код
    - gocritic       # стилистические проверки
    - revive         # замена golint
    - gosec          # security issues
    - prealloc       # предаллокация слайсов
    - noctx          # HTTP requests без context

linters-settings:
  govet:
    enable-all: true
  errcheck:
    check-type-assertions: true
  gocritic:
    enabled-tags:
      - performance
      - diagnostic

issues:
  exclude-use-default: false
  max-issues-per-linter: 0
  max-same-issues: 0
```

```bash
# Установка
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Запуск
golangci-lint run ./...
```

## GitHub Actions

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - uses: golangci/golangci-lint-action@v6
        with:
          version: latest

  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
          POSTGRES_DB: testdb
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - name: Run tests
        env:
          DATABASE_URL: postgres://test:test@localhost:5432/testdb?sslmode=disable
        run: |
          go test -race -coverprofile=coverage.out ./...
          go tool cover -func=coverage.out

  build:
    needs: [lint, test]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - name: Build
        run: |
          CGO_ENABLED=0 go build -ldflags="-s -w" -o server ./cmd/server
      - name: Build Docker
        run: |
          docker build -t myapp:${{ github.sha }} .
```

## Makefile

```makefile
.PHONY: all build test lint run clean

BINARY=server
VERSION=$(shell git describe --tags --always)
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

all: lint test build

build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/server

test:
	go test -race -count=1 ./...

test-integration:
	go test -race -tags integration -count=1 ./...

test-cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

run:
	go run ./cmd/server

generate:
	go generate ./...
	buf generate

migrate-up:
	goose -dir migrations postgres "$$DATABASE_URL" up

migrate-down:
	goose -dir migrations postgres "$$DATABASE_URL" down

docker-build:
	docker build -t myapp:$(VERSION) .

clean:
	rm -rf bin/ coverage.out coverage.html
```

## Vulnerability Scanning

```bash
# Встроенный в Go
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# В CI:
- name: Vulnerability check
  run: govulncheck ./...

# Docker image scanning
docker scout cves myapp:latest
# или
trivy image myapp:latest
```
