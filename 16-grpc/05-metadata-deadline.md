# gRPC: Метаданные и Deadlines

## Metadata (аналог HTTP headers)

```go
import "google.golang.org/grpc/metadata"

// Client → Server: отправка metadata
md := metadata.New(map[string]string{
    "authorization": "Bearer " + token,
    "x-request-id":  uuid.NewString(),
    "x-client-version": "1.2.0",
})
ctx := metadata.NewOutgoingContext(ctx, md)
resp, err := client.GetUser(ctx, req)

// Добавить к существующему context
ctx = metadata.AppendToOutgoingContext(ctx,
    "x-trace-id", traceID,
    "x-user-agent", "my-service/1.0",
)

// Server: чтение metadata
func (s *server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return nil, status.Error(codes.Internal, "no metadata")
    }

    // md — map[string][]string
    tokens := md.Get("authorization")  // ["Bearer xxx"]
    requestIDs := md.Get("x-request-id")

    // Все ключи lowercase (gRPC нормализует)
}
```

### Server → Client: response metadata

```go
// Server: отправить headers (до response)
header := metadata.Pairs(
    "x-request-id", requestID,
    "x-ratelimit-remaining", "42",
)
grpc.SendHeader(ctx, header)

// Server: отправить trailers (после response)
trailer := metadata.Pairs(
    "x-processing-time-ms", fmt.Sprintf("%d", duration.Milliseconds()),
)
grpc.SetTrailer(ctx, trailer)

// Client: получить headers и trailers
var header, trailer metadata.MD
resp, err := client.GetUser(ctx, req,
    grpc.Header(&header),
    grpc.Trailer(&trailer),
)
fmt.Println("Request ID:", header.Get("x-request-id"))
fmt.Println("Processing time:", trailer.Get("x-processing-time-ms"))
```

## Deadlines и Timeouts

```
Deadline = абсолютное время (когда истечёт)
Timeout = относительное время (через сколько)

gRPC передаёт DEADLINE по цепочке сервисов:
  Client (timeout 5s) → Service A (осталось 4.5s) → Service B (осталось 3s)

Это критически важно для microservices:
  Без deadline propagation: A ждёт B, B ждёт C — все ресурсы заняты
  С deadline: timeout каскадно отменяет всю цепочку
```

```go
// Client: установить deadline
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
resp, err := client.GetUser(ctx, req)
if err != nil {
    if status.Code(err) == codes.DeadlineExceeded {
        log.Println("request timed out")
    }
}

// Server: проверить deadline
func (s *server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    // Проверить оставшееся время
    if deadline, ok := ctx.Deadline(); ok {
        remaining := time.Until(deadline)
        if remaining < 100*time.Millisecond {
            return nil, status.Error(codes.DeadlineExceeded, "insufficient time")
        }
    }

    // ctx.Done() будет закрыт при deadline
    select {
    case <-ctx.Done():
        return nil, status.FromContextError(ctx.Err()).Err()
    case result := <-doWork(ctx):
        return result, nil
    }
}

// Server → downstream: передать context (deadline propagation автоматически!)
func (s *server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    // ctx уже содержит deadline от клиента
    // gRPC автоматически передаст его в downstream вызовы
    profile, err := s.profileClient.GetProfile(ctx, &pb.GetProfileRequest{UserId: req.Id})
    // ...
}
```

### Best Practices для Deadlines

```
1. ВСЕГДА ставь deadline на клиенте:
   ❌ client.GetUser(context.Background(), req)
   ✅ ctx, cancel := context.WithTimeout(ctx, 5*time.Second)

2. Выбор timeout:
   - Unary RPC: 1-5s для внутренних, 10-30s для внешних
   - Streaming: timeout на соединение + keepalive
   - Cascade: каждый hop уменьшает timeout

3. Сервер должен проверять ctx.Done():
   - Длинные операции — periodic check
   - DB queries — передавай ctx

4. Не ставь слишком маленький timeout:
   - p99 latency + buffer
   - Учитывай retry + backoff
```

## Keepalive

```go
// Server
server := grpc.NewServer(
    grpc.KeepaliveParams(keepalive.ServerParameters{
        MaxConnectionIdle:     5 * time.Minute,  // закрыть idle connection
        MaxConnectionAge:      30 * time.Minute,  // максимальное время жизни
        MaxConnectionAgeGrace: 5 * time.Second,   // grace period для завершения RPC
        Time:                  1 * time.Minute,   // ping interval
        Timeout:               20 * time.Second,  // ping timeout
    }),
    grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
        MinTime:             10 * time.Second, // минимальный интервал ping от клиента
        PermitWithoutStream: true,             // разрешить ping без активных streams
    }),
)

// Client
conn, _ := grpc.NewClient(addr,
    grpc.WithKeepaliveParams(keepalive.ClientParameters{
        Time:                10 * time.Second, // ping interval
        Timeout:             3 * time.Second,  // ping timeout
        PermitWithoutStream: true,
    }),
)
```

## Частые вопросы

**Q: Metadata vs protobuf поля — когда что?**
A: Metadata — cross-cutting concerns (auth, tracing, request ID). Protobuf поля — бизнес-данные. Аналогия: HTTP headers vs body.

**Q: Что если клиент не ставит deadline?**
A: Сервер может установить свой через `context.WithTimeout`. Но лучше — enforced server-side default deadline через interceptor.

**Q: gRPC timeout vs HTTP timeout?**
A: gRPC deadline передаётся downstream (propagation). HTTP timeout — только между двумя точками. gRPC deadline — мощнее для microservices.
