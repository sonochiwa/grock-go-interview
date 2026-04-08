# gRPC: Error Handling

## Status Codes

```
gRPC код              HTTP аналог    Когда использовать
──────────────────────────────────────────────────────────
OK (0)                200            Успех
Canceled (1)          499            Клиент отменил
Unknown (2)           500            Неизвестная ошибка
InvalidArgument (3)   400            Невалидные параметры
DeadlineExceeded (4)  504            Timeout
NotFound (5)          404            Ресурс не найден
AlreadyExists (6)     409            Ресурс уже существует
PermissionDenied (7)  403            Нет прав
ResourceExhausted (8) 429            Rate limited / квота
FailedPrecondition (9) 400           Предусловие не выполнено
Aborted (10)          409            Конфликт (optimistic lock)
OutOfRange (11)       400            Вне допустимого диапазона
Unimplemented (12)    501            Метод не реализован
Internal (13)         500            Внутренняя ошибка
Unavailable (14)      503            Сервис недоступен (retry!)
DataLoss (15)         500            Потеря данных
Unauthenticated (16)  401            Не аутентифицирован
```

## Базовое использование

```go
import (
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// Создание ошибки
func (s *server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    if req.Id == "" {
        return nil, status.Error(codes.InvalidArgument, "id is required")
    }

    user, err := s.store.Get(ctx, req.Id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, status.Errorf(codes.NotFound, "user %s not found", req.Id)
        }
        // НЕ раскрывать внутренние ошибки клиенту!
        log.Printf("internal error: %v", err)
        return nil, status.Error(codes.Internal, "internal error")
    }

    return &pb.GetUserResponse{User: toProto(user)}, nil
}

// Обработка на клиенте
resp, err := client.GetUser(ctx, req)
if err != nil {
    st, ok := status.FromError(err)
    if !ok {
        // Не gRPC ошибка
        log.Fatal("unexpected error:", err)
    }

    switch st.Code() {
    case codes.NotFound:
        fmt.Println("User not found")
    case codes.InvalidArgument:
        fmt.Println("Bad request:", st.Message())
    case codes.Unavailable:
        // Можно retry
        fmt.Println("Service unavailable, retrying...")
    default:
        fmt.Printf("RPC error: code=%s msg=%s\n", st.Code(), st.Message())
    }
    return
}
```

## Rich Error Details (Error Details API)

```go
import (
    "google.golang.org/genproto/googleapis/rpc/errdetails"
    "google.golang.org/grpc/status"
)

// Server: ошибка с деталями
func (s *server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
    var violations []*errdetails.BadRequest_FieldViolation

    if req.Name == "" {
        violations = append(violations, &errdetails.BadRequest_FieldViolation{
            Field:       "name",
            Description: "name is required",
        })
    }
    if !isValidEmail(req.Email) {
        violations = append(violations, &errdetails.BadRequest_FieldViolation{
            Field:       "email",
            Description: "invalid email format",
        })
    }

    if len(violations) > 0 {
        st := status.New(codes.InvalidArgument, "validation failed")
        st, _ = st.WithDetails(&errdetails.BadRequest{
            FieldViolations: violations,
        })
        return nil, st.Err()
    }

    // ...
}

// Client: извлечение деталей
resp, err := client.CreateUser(ctx, req)
if err != nil {
    st := status.Convert(err)
    for _, detail := range st.Details() {
        switch d := detail.(type) {
        case *errdetails.BadRequest:
            for _, v := range d.FieldViolations {
                fmt.Printf("Field %s: %s\n", v.Field, v.Description)
            }
        case *errdetails.RetryInfo:
            fmt.Printf("Retry after: %s\n", d.RetryDelay.AsDuration())
        case *errdetails.QuotaFailure:
            for _, v := range v.Violations {
                fmt.Printf("Quota: %s — %s\n", v.Subject, v.Description)
            }
        }
    }
}
```

## Типы Error Details (google.rpc)

```
errdetails.BadRequest        — FieldViolations (валидация)
errdetails.RetryInfo         — RetryDelay (когда ретраить)
errdetails.QuotaFailure      — Violations (какая квота)
errdetails.PreconditionFailure — Violations (какое условие)
errdetails.ErrorInfo         — Reason, Domain, Metadata (структурированная ошибка)
errdetails.ResourceInfo      — ResourceType, ResourceName (какой ресурс)
errdetails.DebugInfo         — StackEntries, Detail (только для dev)
errdetails.Help              — Links (ссылки на документацию)
errdetails.LocalizedMessage  — Locale, Message (i18n)
```

## Best Practices

```
1. НЕ раскрывай внутренние ошибки:
   ❌ return nil, status.Errorf(codes.Internal, "pq: duplicate key %v", err)
   ✅ return nil, status.Error(codes.AlreadyExists, "user already exists")

2. Используй правильные коды:
   ❌ codes.Internal для "not found"
   ✅ codes.NotFound

3. Retriable ошибки:
   codes.Unavailable — всегда retry
   codes.DeadlineExceeded — retry с осторожностью
   codes.ResourceExhausted — retry после backoff
   Остальные — обычно НЕ retry

4. Не оборачивай gRPC ошибки:
   ❌ fmt.Errorf("failed: %w", status.Error(codes.NotFound, "..."))
   ✅ Возвращай status.Error напрямую
   (fmt.Errorf оборачивает → status.FromError не сработает)

5. Context errors:
   ctx.Err() == context.Canceled → codes.Canceled
   ctx.Err() == context.DeadlineExceeded → codes.DeadlineExceeded
   gRPC делает это автоматически
```

## Частые вопросы

**Q: PermissionDenied vs Unauthenticated?**
A: Unauthenticated (401) — не знаем кто ты (нет токена). PermissionDenied (403) — знаем кто ты, но нет прав.

**Q: FailedPrecondition vs InvalidArgument?**
A: InvalidArgument — запрос невалиден сам по себе (пустое имя). FailedPrecondition — запрос валиден, но система не в нужном состоянии (удаление непустой папки).

**Q: Как логировать gRPC ошибки?**
A: Logging interceptor. Логировать `codes.Internal` и `codes.Unknown` как ERROR, остальные как INFO/DEBUG.
