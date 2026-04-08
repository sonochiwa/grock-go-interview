# gRPC: Основы и Protocol Buffers

## Что такое gRPC

```
gRPC = Google Remote Procedure Call

Клиент вызывает метод на сервере как будто это локальная функция:
  result, err := client.GetUser(ctx, &pb.GetUserRequest{Id: 123})

Стек:
  [Клиент Go] → [gRPC stub] → [HTTP/2] → [gRPC server] → [Handler Go]
                     ↕                          ↕
              [Protobuf encode]          [Protobuf decode]

Преимущества:
  + Бинарный формат (protobuf) — в 5-10x меньше JSON
  + HTTP/2 — мультиплексирование, сжатие headers
  + Строгая типизация (schema = .proto файл)
  + Кодогенерация клиента и сервера
  + Native streaming (bidirectional)
  + Deadline propagation (timeout по всей цепочке)

Недостатки:
  - Не читаемый человеком (бинарный)
  - Нет нативной поддержки в браузерах (нужен grpc-web)
  - Сложнее дебажить (curl не работает, нужен grpcurl)
  - Tooling меньше чем у REST
```

## Protocol Buffers (Protobuf)

### Определение сообщений

```protobuf
// user.proto
syntax = "proto3";

package user.v1;             // логический namespace
option go_package = "gen/user/v1;userv1";  // Go import path

// Сообщение = структура данных
message User {
  string id = 1;             // 1 = field number (не значение!)
  string name = 2;
  string email = 3;
  UserRole role = 4;
  repeated string tags = 5;  // repeated = slice
  optional string bio = 6;   // optional = pointer (*string)
  google.protobuf.Timestamp created_at = 7;

  // Вложенное сообщение
  Address address = 8;

  // Oneof — только одно поле может быть заполнено
  oneof contact {
    string phone = 9;
    string telegram = 10;
  }
}

message Address {
  string city = 1;
  string street = 2;
  string zip = 3;
}

enum UserRole {
  USER_ROLE_UNSPECIFIED = 0;  // ВСЕГДА 0 = default
  USER_ROLE_ADMIN = 1;
  USER_ROLE_USER = 2;
  USER_ROLE_MODERATOR = 3;
}
```

### Типы данных

```
Protobuf         Go              Default      Wire
─────────────────────────────────────────────────
double           float64          0            fixed 8 bytes
float            float32          0            fixed 4 bytes
int32            int32            0            varint
int64            int64            0            varint
uint32           uint32           0            varint
uint64           uint64           0            varint
sint32           int32            0            zigzag varint (лучше для отрицательных)
sint64           int64            0            zigzag varint
fixed32          uint32           0            fixed 4 bytes (лучше для > 2^28)
fixed64          uint64           0            fixed 8 bytes
bool             bool             false        varint
string           string           ""           length-delimited
bytes            []byte           nil          length-delimited
message          *Message         nil          length-delimited
repeated T       []T              nil          packed
map<K,V>         map[K]V          nil          repeated KV pair
```

### Field Numbers — правила

```
Правила:
  1-15:     1 байт на encoding → используй для частых полей
  16-2047:  2 байта
  19000-19999: зарезервированы (protobuf internal)

Обратная совместимость:
  ✅ Добавить новое поле (старый код игнорирует)
  ✅ Удалить поле (но зарезервировать номер!)
  ❌ Менять тип поля
  ❌ Менять номер поля
  ❌ Переименовать поле (wire format использует номер, не имя)

reserved 6, 15, 9 to 11;        // зарезервировать номера
reserved "old_field", "legacy";  // зарезервировать имена
```

### Well-Known Types

```protobuf
import "google/protobuf/timestamp.proto";
import "google/protobuf/duration.proto";
import "google/protobuf/wrappers.proto";
import "google/protobuf/empty.proto";
import "google/protobuf/any.proto";
import "google/protobuf/struct.proto";  // динамический JSON

// Timestamp → time.Time
google.protobuf.Timestamp created_at = 1;

// Duration → time.Duration
google.protobuf.Duration timeout = 2;

// Wrappers — отличить "не передано" от "default value"
google.protobuf.StringValue nickname = 3;  // → *string
google.protobuf.Int32Value age = 4;        // → *int32

// Empty — для RPC без request/response
google.protobuf.Empty empty = 5;
```

### Определение сервиса

```protobuf
service UserService {
  // Unary RPC
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
  rpc DeleteUser(DeleteUserRequest) returns (google.protobuf.Empty);

  // Server streaming
  rpc ListUsers(ListUsersRequest) returns (stream User);

  // Client streaming
  rpc UploadAvatar(stream UploadAvatarRequest) returns (UploadAvatarResponse);

  // Bidirectional streaming
  rpc Chat(stream ChatMessage) returns (stream ChatMessage);
}

message GetUserRequest {
  string id = 1;
}

message GetUserResponse {
  User user = 1;
}

message ListUsersRequest {
  int32 page_size = 1;
  string page_token = 2;
}

message CreateUserRequest {
  string name = 1;
  string email = 2;
  UserRole role = 3;
}

message CreateUserResponse {
  User user = 1;
}

message DeleteUserRequest {
  string id = 1;
}
```

## Кодогенерация

```bash
# Установка
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Генерация
protoc \
  --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/user/v1/user.proto

# Или с buf (рекомендуется)
# buf.yaml
version: v2
modules:
  - path: proto

# buf.gen.yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen
    opt: paths=source_relative
  - remote: buf.build/grpc/go
    out: gen
    opt: paths=source_relative

buf generate
```

### Сгенерированный код

```go
// Из protoc генерируются:
// 1. user.pb.go — message types
// 2. user_grpc.pb.go — service interface + client

// Серверный интерфейс (нужно реализовать):
type UserServiceServer interface {
    GetUser(context.Context, *GetUserRequest) (*GetUserResponse, error)
    CreateUser(context.Context, *CreateUserRequest) (*CreateUserResponse, error)
    DeleteUser(context.Context, *DeleteUserRequest) (*emptypb.Empty, error)
    ListUsers(*ListUsersRequest, UserService_ListUsersServer) error
    mustEmbedUnimplementedUserServiceServer()
}

// Клиент (готов к использованию):
type UserServiceClient interface {
    GetUser(ctx context.Context, in *GetUserRequest, opts ...grpc.CallOption) (*GetUserResponse, error)
    // ...
}
```

## Реализация сервера

```go
type userServer struct {
    userv1.UnimplementedUserServiceServer // обязательно embed!
    store UserStore
}

func (s *userServer) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
    if req.Id == "" {
        return nil, status.Error(codes.InvalidArgument, "id is required")
    }
    user, err := s.store.Get(ctx, req.Id)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            return nil, status.Error(codes.NotFound, "user not found")
        }
        return nil, status.Error(codes.Internal, "failed to get user")
    }
    return &userv1.GetUserResponse{User: toProto(user)}, nil
}

func main() {
    lis, _ := net.Listen("tcp", ":50051")
    s := grpc.NewServer()
    userv1.RegisterUserServiceServer(s, &userServer{store: newStore()})

    // Graceful shutdown
    go func() {
        sigCh := make(chan os.Signal, 1)
        signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
        <-sigCh
        s.GracefulStop()
    }()

    log.Println("gRPC server listening on :50051")
    if err := s.Serve(lis); err != nil {
        log.Fatal(err)
    }
}
```

## Реализация клиента

```go
func main() {
    conn, err := grpc.NewClient("localhost:50051",
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    client := userv1.NewUserServiceClient(conn)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    resp, err := client.GetUser(ctx, &userv1.GetUserRequest{Id: "123"})
    if err != nil {
        st, ok := status.FromError(err)
        if ok {
            log.Printf("gRPC error: code=%s, msg=%s", st.Code(), st.Message())
        }
        log.Fatal(err)
    }
    fmt.Printf("User: %+v\n", resp.User)
}
```

## Частые вопросы

**Q: Почему protobuf быстрее JSON?**
A: Бинарный формат (varint encoding), нет парсинга строк, field numbers вместо имён, нет лишних символов ({}, "", :). Меньше данных на wire + быстрее encode/decode.

**Q: Зачем UnimplementedServer?**
A: Forward compatibility. Если добавить новый RPC в .proto, сервер не сломается — UnimplementedServer вернёт `codes.Unimplemented`. Без него — ошибка компиляции.

**Q: proto2 vs proto3?**
A: proto3 — всегда используй для новых проектов. Упрощённый: нет required/optional (всё optional), default values = zero values, нет extensions.
