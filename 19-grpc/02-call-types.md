# gRPC: Типы вызовов

## Обзор

```
4 типа RPC:

1. Unary:           Client --1 request--> Server --1 response-->
2. Server Streaming: Client --1 request--> Server --N responses-->
3. Client Streaming: Client --N requests--> Server --1 response-->
4. Bidirectional:   Client --N requests--> Server --N responses-->
```

## 1. Unary RPC

```protobuf
rpc GetUser(GetUserRequest) returns (GetUserResponse);
```

```go
// Server
func (s *server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    user, err := s.store.Get(ctx, req.Id)
    if err != nil {
        return nil, status.Errorf(codes.NotFound, "user %s not found", req.Id)
    }
    return &pb.GetUserResponse{User: user}, nil
}

// Client
resp, err := client.GetUser(ctx, &pb.GetUserRequest{Id: "123"})
```

## 2. Server Streaming

```
Когда: большие результаты, real-time updates, long polling замена
Примеры: список результатов поиска, live logs, price updates
```

```protobuf
rpc ListUsers(ListUsersRequest) returns (stream User);
rpc WatchOrders(WatchOrdersRequest) returns (stream OrderUpdate);
```

```go
// Server
func (s *server) ListUsers(req *pb.ListUsersRequest, stream pb.UserService_ListUsersServer) error {
    users, err := s.store.List(stream.Context(), req.Filter)
    if err != nil {
        return status.Error(codes.Internal, "failed to list users")
    }
    for _, user := range users {
        if err := stream.Send(user); err != nil {
            return err // client disconnected
        }
    }
    return nil // закрывает stream
}

// Client
stream, err := client.ListUsers(ctx, &pb.ListUsersRequest{Filter: "active"})
if err != nil {
    log.Fatal(err)
}
for {
    user, err := stream.Recv()
    if err == io.EOF {
        break // stream закончился
    }
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("User: %+v\n", user)
}
```

## 3. Client Streaming

```
Когда: upload файлов, batch операции, агрегация данных
Примеры: upload изображения chunks, отправка метрик
```

```protobuf
rpc UploadAvatar(stream UploadAvatarRequest) returns (UploadAvatarResponse);
```

```go
// Server
func (s *server) UploadAvatar(stream pb.UserService_UploadAvatarServer) error {
    var buf bytes.Buffer
    var userID string

    for {
        chunk, err := stream.Recv()
        if err == io.EOF {
            // Все chunks получены
            url, err := s.storage.Save(userID, buf.Bytes())
            if err != nil {
                return status.Error(codes.Internal, "failed to save")
            }
            return stream.SendAndClose(&pb.UploadAvatarResponse{
                Url:  url,
                Size: int64(buf.Len()),
            })
        }
        if err != nil {
            return err
        }
        if userID == "" {
            userID = chunk.UserId
        }
        buf.Write(chunk.Data)
    }
}

// Client
stream, err := client.UploadAvatar(ctx)
if err != nil {
    log.Fatal(err)
}

// Отправляем файл chunks по 64KB
buf := make([]byte, 64*1024)
for {
    n, err := file.Read(buf)
    if err == io.EOF {
        break
    }
    if err := stream.Send(&pb.UploadAvatarRequest{
        UserId: "123",
        Data:   buf[:n],
    }); err != nil {
        log.Fatal(err)
    }
}

resp, err := stream.CloseAndRecv()
fmt.Printf("Uploaded: %s (%d bytes)\n", resp.Url, resp.Size)
```

## 4. Bidirectional Streaming

```
Когда: chat, real-time sync, interactive processing
Примеры: чат, совместное редактирование, game state
```

```protobuf
rpc Chat(stream ChatMessage) returns (stream ChatMessage);
```

```go
// Server
func (s *server) Chat(stream pb.ChatService_ChatServer) error {
    for {
        msg, err := stream.Recv()
        if err == io.EOF {
            return nil
        }
        if err != nil {
            return err
        }

        // Обработка и отправка ответа
        reply := &pb.ChatMessage{
            UserId:  "system",
            Content: "Echo: " + msg.Content,
        }
        if err := stream.Send(reply); err != nil {
            return err
        }
    }
}

// Client
stream, err := client.Chat(ctx)
if err != nil {
    log.Fatal(err)
}

// Отправка в отдельной горутине
go func() {
    for _, text := range messages {
        if err := stream.Send(&pb.ChatMessage{
            UserId:  "user1",
            Content: text,
        }); err != nil {
            log.Fatal(err)
        }
    }
    stream.CloseSend() // закрываем отправку
}()

// Получение
for {
    msg, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("[%s]: %s\n", msg.UserId, msg.Content)
}
```

## Сравнение

```
| Тип | Request | Response | Use Case |
|---|---|---|---|
| Unary | 1 | 1 | CRUD, простые запросы |
| Server stream | 1 | N | Списки, подписки, live data |
| Client stream | N | 1 | Upload, batch, агрегация |
| Bidi stream | N | N | Chat, sync, interactive |
```

## Частые вопросы

**Q: Когда streaming вместо unary с пагинацией?**
A: Streaming — когда данные генерируются постепенно, нужен real-time, или данных очень много. Пагинация — когда клиент сам контролирует скорость и нужен random access к страницам.

**Q: Что если stream оборвётся?**
A: gRPC вернёт ошибку при следующем Send/Recv. Клиент должен переподключиться и, возможно, продолжить с последней позиции (нужна своя логика).

**Q: Можно ли отменить streaming?**
A: Да, через context cancellation. `cancel()` на клиенте → сервер получит ошибку при Send/Recv.
