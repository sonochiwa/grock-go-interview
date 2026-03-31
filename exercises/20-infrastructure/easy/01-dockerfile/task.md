# Dockerfile

Это задание без кода — напиши оптимальный multi-stage Dockerfile.

Требования:
1. Stage 1 (builder): `golang:1.24-alpine`, кэшируй `go mod download`, собери бинарник с `-ldflags="-s -w"` и `CGO_ENABLED=0`
2. Stage 2 (runtime): `gcr.io/distroless/static-debian12:nonroot`
3. `USER nonroot:nonroot`
4. `EXPOSE 8080`

Бонус: добавь version injection через `--build-arg VERSION=...`

Открой `main.go` — там stub Dockerfile в комментарии. Перенеси в `Dockerfile`.
