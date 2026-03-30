# gRPC: Interceptors и Middleware

## Обзор

```
Interceptor = middleware для gRPC
Выполняется до/после каждого RPC вызова

4 типа:
  - UnaryServerInterceptor
  - StreamServerInterceptor
  - UnaryClientInterceptor
  - StreamClientInterceptor

Типичное использование:
  - Logging
  - Metrics (Prometheus)
  - Authentication/Authorization
  - Recovery (panic → error)
  - Tracing (OpenTelemetry)
  - Validation
  - Rate limiting
```

## Server Interceptors

### Unary Server Interceptor

```go
// Сигнатура
type UnaryServerInterceptor func(
    ctx context.Context,
    req any,
    info *grpc.UnaryServerInfo,  // Method name, server
    handler grpc.UnaryHandler,    // следующий handler
) (any, error)

// Logging interceptor
func loggingInterceptor(
    ctx context.Context,
    req any,
    info *grpc.UnaryServerInfo,
    handler grpc.UnaryHandler,
) (any, error) {
    start := time.Now()

    // Вызов handler
    resp, err := handler(ctx, req)

    // После вызова
    duration := time.Since(start)
    code := status.Code(err)
    log.Printf("method=%s duration=%s code=%s",
        info.FullMethod, duration, code)

    return resp, err
}
```

### Recovery Interceptor

```go
func recoveryInterceptor(
    ctx context.Context,
    req any,
    info *grpc.UnaryServerInfo,
    handler grpc.UnaryHandler,
) (resp any, err error) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("panic recovered: %v\nstack: %s", r, debug.Stack())
            err = status.Errorf(codes.Internal, "internal error")
        }
    }()
    return handler(ctx, req)
}
```

### Auth Interceptor

```go
func authInterceptor(
    ctx context.Context,
    req any,
    info *grpc.UnaryServerInfo,
    handler grpc.UnaryHandler,
) (any, error) {
    // Пропустить health check
    if info.FullMethod == "/grpc.health.v1.Health/Check" {
        return handler(ctx, req)
    }

    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "no metadata")
    }

    tokens := md.Get("authorization")
    if len(tokens) == 0 {
        return nil, status.Error(codes.Unauthenticated, "no token")
    }

    userID, err := validateToken(tokens[0])
    if err != nil {
        return nil, status.Error(codes.Unauthenticated, "invalid token")
    }

    // Добавить userID в context
    ctx = context.WithValue(ctx, userIDKey, userID)
    return handler(ctx, req)
}
```

### Stream Server Interceptor

```go
type StreamServerInterceptor func(
    srv any,
    ss grpc.ServerStream,
    info *grpc.StreamServerInfo,
    handler grpc.StreamHandler,
) error

func streamLoggingInterceptor(
    srv any,
    ss grpc.ServerStream,
    info *grpc.StreamServerInfo,
    handler grpc.StreamHandler,
) error {
    start := time.Now()
    err := handler(srv, ss)
    log.Printf("stream method=%s duration=%s err=%v",
        info.FullMethod, time.Since(start), err)
    return err
}
```

## Client Interceptors

```go
func clientLoggingInterceptor(
    ctx context.Context,
    method string,
    req, reply any,
    cc *grpc.ClientConn,
    invoker grpc.UnaryInvoker,
    opts ...grpc.CallOption,
) error {
    start := time.Now()
    err := invoker(ctx, method, req, reply, cc, opts...)
    log.Printf("client call method=%s duration=%s err=%v",
        method, time.Since(start), err)
    return err
}

// Retry interceptor
func retryInterceptor(maxRetries int) grpc.UnaryClientInterceptor {
    return func(ctx context.Context, method string, req, reply any,
        cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
        var lastErr error
        for i := 0; i <= maxRetries; i++ {
            lastErr = invoker(ctx, method, req, reply, cc, opts...)
            if lastErr == nil {
                return nil
            }
            code := status.Code(lastErr)
            // Ретраить только retriable ошибки
            if code != codes.Unavailable && code != codes.DeadlineExceeded {
                return lastErr
            }
            time.Sleep(time.Duration(1<<i) * 100 * time.Millisecond)
        }
        return lastErr
    }
}
```

## Chaining (несколько interceptors)

```go
// Server — interceptors выполняются в порядке передачи
server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        recoveryInterceptor,     // 1. Ловит panic
        loggingInterceptor,      // 2. Логирует
        metricsInterceptor,      // 3. Метрики
        authInterceptor,         // 4. Аутентификация
        validationInterceptor,   // 5. Валидация
    ),
    grpc.ChainStreamInterceptor(
        streamRecoveryInterceptor,
        streamLoggingInterceptor,
    ),
)

// Client
conn, _ := grpc.NewClient(addr,
    grpc.WithChainUnaryInterceptor(
        clientLoggingInterceptor,
        retryInterceptor(3),
    ),
)
```

## go-grpc-middleware (готовые interceptors)

```go
import (
    "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
    "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
    "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
    "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/ratelimit"
    "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/validator"
)

server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        recovery.UnaryServerInterceptor(),
        logging.UnaryServerInterceptor(logger),
        auth.UnaryServerInterceptor(authFunc),
        validator.UnaryServerInterceptor(),
    ),
)
```

## Частые вопросы

**Q: Interceptor vs Middleware — разница?**
A: В gRPC interceptor — это и есть middleware. Термин "interceptor" используется в gRPC, "middleware" — в HTTP. Концепция одна — обработка до/после основного handler.

**Q: Порядок interceptors важен?**
A: Да! Recovery должен быть первым (обёрнуть всё). Auth — перед бизнес-логикой. Logging — один из первых (чтобы залогировать всё).

**Q: Как передать данные между interceptors?**
A: Через `context.WithValue`. Auth interceptor добавляет userID → бизнес-логика читает из context.
