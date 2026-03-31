package dockerfile

// Solution Dockerfile:
//
// # Stage 1: Build
// FROM golang:1.24-alpine AS builder
//
// ARG VERSION=dev
//
// WORKDIR /app
//
// COPY go.mod go.sum ./
// RUN go mod download
//
// COPY . .
// RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
//     go build -ldflags="-s -w -X main.version=${VERSION}" \
//     -o /app/server ./cmd/server
//
// # Stage 2: Runtime
// FROM gcr.io/distroless/static-debian12:nonroot
//
// COPY --from=builder /app/server /server
//
// USER nonroot:nonroot
// EXPOSE 8080
//
// ENTRYPOINT ["/server"]
//
// # Build: docker build --build-arg VERSION=v1.0.0 -t myapp .
// # Size: ~12-15MB
