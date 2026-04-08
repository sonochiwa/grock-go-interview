# MongoDB: Паттерны в Go

## Repository Pattern

### Интерфейс

```go
type UserRepository interface {
    GetByID(ctx context.Context, id primitive.ObjectID) (User, error)
    GetByEmail(ctx context.Context, email string) (User, error)
    List(ctx context.Context, filter UserFilter) (Page[User], error)
    Create(ctx context.Context, user *User) error
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id primitive.ObjectID) error
}

type User struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    Name      string             `bson:"name"          json:"name"`
    Email     string             `bson:"email"         json:"email"`
    Status    string             `bson:"status"        json:"status"`
    Version   int                `bson:"version"       json:"version"`
    CreatedAt time.Time          `bson:"created_at"    json:"created_at"`
    UpdatedAt time.Time          `bson:"updated_at"    json:"updated_at"`
    DeletedAt *time.Time         `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

type UserFilter struct {
    Status string
    Name   string // partial match
    Limit  int64
    Cursor string // cursor-based pagination
}
```

### Реализация

```go
var (
    ErrNotFound      = errors.New("not found")
    ErrAlreadyExists = errors.New("already exists")
    ErrConflict      = errors.New("version conflict")
)

type mongoUserRepo struct {
    coll *mongo.Collection
}

func NewUserRepository(db *mongo.Database) UserRepository {
    return &mongoUserRepo{
        coll: db.Collection("users"),
    }
}

func (r *mongoUserRepo) GetByID(ctx context.Context, id primitive.ObjectID) (User, error) {
    var user User
    filter := bson.M{
        "_id":        id,
        "deleted_at": bson.M{"$exists": false},
    }
    err := r.coll.FindOne(ctx, filter).Decode(&user)
    if errors.Is(err, mongo.ErrNoDocuments) {
        return User{}, ErrNotFound
    }
    return user, err
}

func (r *mongoUserRepo) GetByEmail(ctx context.Context, email string) (User, error) {
    var user User
    filter := bson.M{
        "email":      email,
        "deleted_at": bson.M{"$exists": false},
    }
    err := r.coll.FindOne(ctx, filter).Decode(&user)
    if errors.Is(err, mongo.ErrNoDocuments) {
        return User{}, ErrNotFound
    }
    return user, err
}

func (r *mongoUserRepo) Create(ctx context.Context, user *User) error {
    now := time.Now()
    user.CreatedAt = now
    user.UpdatedAt = now
    user.Version = 1
    user.Status = "active"

    result, err := r.coll.InsertOne(ctx, user)
    if err != nil {
        if mongo.IsDuplicateKeyError(err) {
            return ErrAlreadyExists
        }
        return fmt.Errorf("insert user: %w", err)
    }
    user.ID = result.InsertedID.(primitive.ObjectID)
    return nil
}

func (r *mongoUserRepo) Update(ctx context.Context, user *User) error {
    now := time.Now()

    // Optimistic locking via version field
    filter := bson.M{
        "_id":        user.ID,
        "version":    user.Version,
        "deleted_at": bson.M{"$exists": false},
    }
    update := bson.M{
        "$set": bson.M{
            "name":       user.Name,
            "email":      user.Email,
            "status":     user.Status,
            "updated_at": now,
        },
        "$inc": bson.M{"version": 1},
    }

    result, err := r.coll.UpdateOne(ctx, filter, update)
    if err != nil {
        if mongo.IsDuplicateKeyError(err) {
            return ErrAlreadyExists
        }
        return fmt.Errorf("update user: %w", err)
    }
    if result.MatchedCount == 0 {
        return ErrConflict
    }

    user.Version++
    user.UpdatedAt = now
    return nil
}

func (r *mongoUserRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
    // Soft delete — see next section
    now := time.Now()
    filter := bson.M{
        "_id":        id,
        "deleted_at": bson.M{"$exists": false},
    }
    update := bson.M{
        "$set": bson.M{"deleted_at": now, "updated_at": now},
    }

    result, err := r.coll.UpdateOne(ctx, filter, update)
    if err != nil {
        return fmt.Errorf("delete user: %w", err)
    }
    if result.MatchedCount == 0 {
        return ErrNotFound
    }
    return nil
}
```

### Использование в сервисном слое

```go
type UserService struct {
    users UserRepository
}

func NewUserService(users UserRepository) *UserService {
    return &UserService{users: users}
}

func (s *UserService) Register(ctx context.Context, name, email string) (*User, error) {
    user := &User{
        Name:  name,
        Email: email,
    }
    if err := s.users.Create(ctx, user); err != nil {
        return nil, fmt.Errorf("register user: %w", err)
    }
    return user, nil
}
```

## Soft Delete

### Реализация

Мягкое удаление — пометка `deleted_at` вместо физического удаления. Позволяет восстановить данные и вести аудит.

```go
// Все запросы к "живым" документам включают фильтр
func activeFilter(extra bson.M) bson.M {
    filter := bson.M{"deleted_at": bson.M{"$exists": false}}
    for k, v := range extra {
        filter[k] = v
    }
    return filter
}

func (r *mongoUserRepo) GetByID(ctx context.Context, id primitive.ObjectID) (User, error) {
    var user User
    err := r.coll.FindOne(ctx, activeFilter(bson.M{"_id": id})).Decode(&user)
    if errors.Is(err, mongo.ErrNoDocuments) {
        return User{}, ErrNotFound
    }
    return user, err
}

func (r *mongoUserRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
    now := time.Now()
    result, err := r.coll.UpdateOne(ctx,
        activeFilter(bson.M{"_id": id}),
        bson.M{"$set": bson.M{"deleted_at": now}},
    )
    if err != nil {
        return err
    }
    if result.MatchedCount == 0 {
        return ErrNotFound
    }
    return nil
}

// Восстановление
func (r *mongoUserRepo) Restore(ctx context.Context, id primitive.ObjectID) error {
    result, err := r.coll.UpdateOne(ctx,
        bson.M{"_id": id, "deleted_at": bson.M{"$exists": true}},
        bson.M{"$unset": bson.M{"deleted_at": ""}},
    )
    if err != nil {
        return err
    }
    if result.MatchedCount == 0 {
        return ErrNotFound
    }
    return nil
}

// Физическое удаление старых soft-deleted записей (cron job)
func (r *mongoUserRepo) PurgeDeleted(ctx context.Context, olderThan time.Time) (int64, error) {
    result, err := r.coll.DeleteMany(ctx, bson.M{
        "deleted_at": bson.M{"$lt": olderThan},
    })
    if err != nil {
        return 0, err
    }
    return result.DeletedCount, nil
}
```

### Индексы для soft delete

```go
// Partial index — только по активным документам
indexModel := mongo.IndexModel{
    Keys: bson.D{{"email", 1}},
    Options: options.Index().
        SetUnique(true).
        SetPartialFilterExpression(bson.M{
            "deleted_at": bson.M{"$exists": false},
        }),
}

// Индекс для purge job
indexModel2 := mongo.IndexModel{
    Keys: bson.D{{"deleted_at", 1}},
    Options: options.Index().
        SetSparse(true), // только документы с deleted_at
}
```

## Cursor-Based Pagination

Offset-пагинация деградирует на больших коллекциях. Cursor-пагинация всегда O(limit) при наличии индекса.

### Реализация

```go
type Page[T any] struct {
    Items      []T    `json:"items"`
    NextCursor string `json:"next_cursor,omitempty"`
    HasMore    bool   `json:"has_more"`
}

type Cursor struct {
    CreatedAt time.Time          `json:"c"`
    ID        primitive.ObjectID `json:"i"`
}

func encodeCursor(createdAt time.Time, id primitive.ObjectID) string {
    c := Cursor{CreatedAt: createdAt, ID: id}
    data, _ := json.Marshal(c)
    return base64.URLEncoding.EncodeToString(data)
}

func decodeCursor(s string) (Cursor, error) {
    data, err := base64.URLEncoding.DecodeString(s)
    if err != nil {
        return Cursor{}, fmt.Errorf("invalid cursor: %w", err)
    }
    var c Cursor
    if err := json.Unmarshal(data, &c); err != nil {
        return Cursor{}, fmt.Errorf("invalid cursor data: %w", err)
    }
    return c, nil
}

func (r *mongoUserRepo) List(ctx context.Context, f UserFilter) (Page[User], error) {
    limit := f.Limit
    if limit <= 0 || limit > 100 {
        limit = 20
    }

    // Base filter: only active documents
    filter := bson.M{"deleted_at": bson.M{"$exists": false}}

    // Optional filters
    if f.Status != "" {
        filter["status"] = f.Status
    }
    if f.Name != "" {
        filter["name"] = bson.M{"$regex": f.Name, "$options": "i"}
    }

    // Cursor filter
    if f.Cursor != "" {
        cur, err := decodeCursor(f.Cursor)
        if err != nil {
            return Page[User]{}, err
        }
        // Documents AFTER the cursor (sorted by created_at DESC, _id DESC)
        filter["$or"] = bson.A{
            bson.M{"created_at": bson.M{"$lt": cur.CreatedAt}},
            bson.M{
                "created_at": cur.CreatedAt,
                "_id":        bson.M{"$lt": cur.ID},
            },
        }
    }

    // Fetch limit+1 to detect if there are more pages
    opts := options.Find().
        SetSort(bson.D{{"created_at", -1}, {"_id", -1}}).
        SetLimit(limit + 1)

    cursor, err := r.coll.Find(ctx, filter, opts)
    if err != nil {
        return Page[User]{}, fmt.Errorf("find users: %w", err)
    }
    defer cursor.Close(ctx)

    var users []User
    if err := cursor.All(ctx, &users); err != nil {
        return Page[User]{}, err
    }

    page := Page[User]{}
    if int64(len(users)) > limit {
        page.HasMore = true
        users = users[:limit]
        last := users[len(users)-1]
        page.NextCursor = encodeCursor(last.CreatedAt, last.ID)
    }
    page.Items = users
    return page, nil
}
```

### Индекс для cursor pagination

```go
// Составной индекс для эффективной cursor-пагинации
indexModel := mongo.IndexModel{
    Keys: bson.D{{"created_at", -1}, {"_id", -1}},
}
```

### HTTP handler

```go
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
    filter := UserFilter{
        Status: r.URL.Query().Get("status"),
        Name:   r.URL.Query().Get("name"),
        Cursor: r.URL.Query().Get("cursor"),
        Limit:  20,
    }

    page, err := h.users.List(r.Context(), filter)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(page)
}

// Response:
// {
//   "items": [...],
//   "next_cursor": "eyJjIjoiMjAyNS0wMS0xNVQxMDozMDowMFoiLCJpIjoiNjc4OTBhYmMifQ==",
//   "has_more": true
// }
//
// Следующая страница: GET /users?cursor=eyJjIjoiMjAyNS0wMS0xNVQxMDozMDowMFoiLCJpIjoiNjc4OTBhYmMifQ==
```

## Change Streams

Change streams позволяют получать уведомления об изменениях в коллекции в реальном времени. Построены поверх oplog.

### Базовый пример

```go
func watchUsers(ctx context.Context, coll *mongo.Collection) error {
    // Следить за insert и update событиями
    pipeline := bson.A{
        bson.M{"$match": bson.M{
            "operationType": bson.M{"$in": bson.A{"insert", "update", "replace"}},
        }},
    }

    opts := options.ChangeStream().
        SetFullDocument(options.UpdateLookup) // include full document in update events

    stream, err := coll.Watch(ctx, pipeline, opts)
    if err != nil {
        return fmt.Errorf("watch: %w", err)
    }
    defer stream.Close(ctx)

    for stream.Next(ctx) {
        var event bson.M
        if err := stream.Decode(&event); err != nil {
            log.Printf("decode error: %v", err)
            continue
        }

        opType := event["operationType"].(string)
        log.Printf("operation: %s, document: %v", opType, event["fullDocument"])
    }

    return stream.Err()
}
```

### Типизированные события

```go
type ChangeEvent struct {
    OperationType string             `bson:"operationType"`
    DocumentKey   DocumentKey        `bson:"documentKey"`
    FullDocument  *User              `bson:"fullDocument"`
    UpdateDesc    *UpdateDescription `bson:"updateDescription"`
    ClusterTime   primitive.Timestamp `bson:"clusterTime"`
    ResumeToken   bson.Raw           `bson:"_id"`
}

type DocumentKey struct {
    ID primitive.ObjectID `bson:"_id"`
}

type UpdateDescription struct {
    UpdatedFields bson.M   `bson:"updatedFields"`
    RemovedFields []string `bson:"removedFields"`
}

func watchUsersTyped(ctx context.Context, coll *mongo.Collection) error {
    stream, err := coll.Watch(ctx, bson.A{},
        options.ChangeStream().SetFullDocument(options.UpdateLookup))
    if err != nil {
        return err
    }
    defer stream.Close(ctx)

    for stream.Next(ctx) {
        var event ChangeEvent
        if err := stream.Decode(&event); err != nil {
            log.Printf("decode: %v", err)
            continue
        }

        switch event.OperationType {
        case "insert":
            log.Printf("new user: %s", event.FullDocument.Email)
        case "update":
            log.Printf("updated user %s: fields=%v",
                event.DocumentKey.ID.Hex(), event.UpdateDesc.UpdatedFields)
        case "delete":
            log.Printf("deleted user: %s", event.DocumentKey.ID.Hex())
        }
    }
    return stream.Err()
}
```

### Resume Token — продолжение после разрыва

```go
func watchWithResume(ctx context.Context, coll *mongo.Collection) error {
    var resumeToken bson.Raw

    // Load resume token from persistent storage (Redis, file, etc.)
    resumeToken = loadResumeToken()

    opts := options.ChangeStream().SetFullDocument(options.UpdateLookup)
    if resumeToken != nil {
        opts.SetResumeAfter(resumeToken)
    }

    stream, err := coll.Watch(ctx, bson.A{}, opts)
    if err != nil {
        return err
    }
    defer stream.Close(ctx)

    for stream.Next(ctx) {
        var event ChangeEvent
        if err := stream.Decode(&event); err != nil {
            log.Printf("decode: %v", err)
            continue
        }

        // Process event...
        processEvent(event)

        // Persist resume token after successful processing
        saveResumeToken(stream.ResumeToken())
    }

    return stream.Err()
}
```

### Pre/Post Image (MongoDB 6.0+)

```go
// Для получения документа ДО изменения (pre-image)
// Требуется включить на коллекции:
// db.createCollection("users", { changeStreamPreAndPostImages: { enabled: true } })

opts := options.ChangeStream().
    SetFullDocument(options.UpdateLookup).         // document AFTER change
    SetFullDocumentBeforeChange(options.WhenAvailable) // document BEFORE change
```

## Тестирование с testcontainers-go

### Setup

```go
import (
    "context"
    "testing"

    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/mongodb"
    "go.mongodb.org/mongo-driver/v2/mongo"
    "go.mongodb.org/mongo-driver/v2/mongo/options"
)

func setupMongoDB(t *testing.T) (*mongo.Client, func()) {
    t.Helper()
    ctx := context.Background()

    // Start MongoDB container with replica set (needed for transactions)
    container, err := mongodb.Run(ctx,
        "mongo:7",
        mongodb.WithReplicaSet("rs0"),
    )
    if err != nil {
        t.Fatalf("start container: %v", err)
    }

    // Get connection string
    uri, err := container.ConnectionString(ctx)
    if err != nil {
        t.Fatalf("connection string: %v", err)
    }

    // Connect
    client, err := mongo.Connect(options.Client().ApplyURI(uri))
    if err != nil {
        t.Fatalf("connect: %v", err)
    }

    // Ping to ensure connection
    if err := client.Ping(ctx, nil); err != nil {
        t.Fatalf("ping: %v", err)
    }

    // Cleanup function
    cleanup := func() {
        client.Disconnect(ctx)
        container.Terminate(ctx)
    }

    return client, cleanup
}
```

### Integration тест для repository

```go
func TestUserRepository_CRUD(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    client, cleanup := setupMongoDB(t)
    defer cleanup()

    // Use unique database per test to avoid interference
    db := client.Database("test_" + t.Name())
    defer db.Drop(context.Background())

    repo := NewUserRepository(db)
    ctx := context.Background()

    // Create
    user := &User{
        Name:  "Alice",
        Email: "alice@example.com",
    }
    err := repo.Create(ctx, user)
    require.NoError(t, err)
    assert.NotEqual(t, primitive.NilObjectID, user.ID)
    assert.Equal(t, 1, user.Version)

    // GetByID
    found, err := repo.GetByID(ctx, user.ID)
    require.NoError(t, err)
    assert.Equal(t, "Alice", found.Name)
    assert.Equal(t, "alice@example.com", found.Email)

    // GetByEmail
    found, err = repo.GetByEmail(ctx, "alice@example.com")
    require.NoError(t, err)
    assert.Equal(t, user.ID, found.ID)

    // Update
    user.Name = "Alice Updated"
    err = repo.Update(ctx, user)
    require.NoError(t, err)
    assert.Equal(t, 2, user.Version)

    // Verify update
    found, err = repo.GetByID(ctx, user.ID)
    require.NoError(t, err)
    assert.Equal(t, "Alice Updated", found.Name)

    // Optimistic lock conflict
    user.Version = 1 // stale version
    err = repo.Update(ctx, user)
    assert.ErrorIs(t, err, ErrConflict)

    // Delete (soft)
    err = repo.Delete(ctx, found.ID)
    require.NoError(t, err)

    // After soft delete — not found
    _, err = repo.GetByID(ctx, found.ID)
    assert.ErrorIs(t, err, ErrNotFound)
}
```

### Тест пагинации

```go
func TestUserRepository_Pagination(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    client, cleanup := setupMongoDB(t)
    defer cleanup()

    db := client.Database("test_pagination")
    defer db.Drop(context.Background())

    repo := NewUserRepository(db)
    ctx := context.Background()

    // Create 25 users
    for i := 0; i < 25; i++ {
        user := &User{
            Name:  fmt.Sprintf("User %02d", i),
            Email: fmt.Sprintf("user%02d@example.com", i),
        }
        require.NoError(t, repo.Create(ctx, user))
        // Small delay for unique created_at
        time.Sleep(time.Millisecond)
    }

    // First page
    page1, err := repo.List(ctx, UserFilter{Limit: 10})
    require.NoError(t, err)
    assert.Len(t, page1.Items, 10)
    assert.True(t, page1.HasMore)
    assert.NotEmpty(t, page1.NextCursor)

    // Second page
    page2, err := repo.List(ctx, UserFilter{Limit: 10, Cursor: page1.NextCursor})
    require.NoError(t, err)
    assert.Len(t, page2.Items, 10)
    assert.True(t, page2.HasMore)

    // Third page (last 5)
    page3, err := repo.List(ctx, UserFilter{Limit: 10, Cursor: page2.NextCursor})
    require.NoError(t, err)
    assert.Len(t, page3.Items, 5)
    assert.False(t, page3.HasMore)
    assert.Empty(t, page3.NextCursor)

    // No duplicates between pages
    seen := make(map[primitive.ObjectID]bool)
    for _, pages := range []Page[User]{page1, page2, page3} {
        for _, u := range pages.Items {
            assert.False(t, seen[u.ID], "duplicate user: %s", u.ID.Hex())
            seen[u.ID] = true
        }
    }
    assert.Len(t, seen, 25)
}
```

### Тест дубликатов (unique constraint)

```go
func TestUserRepository_DuplicateEmail(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    client, cleanup := setupMongoDB(t)
    defer cleanup()

    db := client.Database("test_duplicate")
    defer db.Drop(context.Background())

    repo := NewUserRepository(db)
    ctx := context.Background()

    // Setup unique index
    _, err := db.Collection("users").Indexes().CreateOne(ctx, mongo.IndexModel{
        Keys:    bson.D{{"email", 1}},
        Options: options.Index().SetUnique(true).SetPartialFilterExpression(
            bson.M{"deleted_at": bson.M{"$exists": false}},
        ),
    })
    require.NoError(t, err)

    // Create first user
    user1 := &User{Name: "Alice", Email: "alice@example.com"}
    require.NoError(t, repo.Create(ctx, user1))

    // Attempt duplicate
    user2 := &User{Name: "Alice 2", Email: "alice@example.com"}
    err = repo.Create(ctx, user2)
    assert.ErrorIs(t, err, ErrAlreadyExists)
}
```

### TestMain с shared container

Для ускорения тестов — один контейнер на весь пакет:

```go
var testClient *mongo.Client

func TestMain(m *testing.M) {
    ctx := context.Background()

    container, err := mongodb.Run(ctx, "mongo:7", mongodb.WithReplicaSet("rs0"))
    if err != nil {
        log.Fatalf("start mongodb: %v", err)
    }

    uri, _ := container.ConnectionString(ctx)
    testClient, err = mongo.Connect(options.Client().ApplyURI(uri))
    if err != nil {
        log.Fatalf("connect: %v", err)
    }

    code := m.Run()

    testClient.Disconnect(ctx)
    container.Terminate(ctx)
    os.Exit(code)
}

// Each test uses a unique database
func testDB(t *testing.T) *mongo.Database {
    t.Helper()
    dbName := strings.ReplaceAll(t.Name(), "/", "_")
    db := testClient.Database(dbName)
    t.Cleanup(func() { db.Drop(context.Background()) })
    return db
}
```

## Setup Indexes при старте

```go
// Создание индексов при инициализации приложения
func SetupIndexes(ctx context.Context, db *mongo.Database) error {
    users := db.Collection("users")
    _, err := users.Indexes().CreateMany(ctx, []mongo.IndexModel{
        {
            Keys: bson.D{{"email", 1}},
            Options: options.Index().
                SetUnique(true).
                SetName("idx_email_unique").
                SetPartialFilterExpression(bson.M{
                    "deleted_at": bson.M{"$exists": false},
                }),
        },
        {
            Keys:    bson.D{{"created_at", -1}, {"_id", -1}},
            Options: options.Index().SetName("idx_created_id_pagination"),
        },
        {
            Keys:    bson.D{{"status", 1}, {"created_at", -1}},
            Options: options.Index().SetName("idx_status_created"),
        },
        {
            Keys:    bson.D{{"deleted_at", 1}},
            Options: options.Index().SetSparse(true).SetName("idx_deleted_sparse"),
        },
    })
    return err
}
```

## Типичные ошибки

```
1. Repository возвращает *mongo.Collection вместо интерфейса
   Нарушает инверсию зависимостей, невозможно мокать
   РЕШЕНИЕ: определять интерфейс на стороне потребителя

2. Soft delete без фильтра в каждом запросе
   Забыли добавить "deleted_at": {"$exists": false} → видны удалённые
   РЕШЕНИЕ: helper-функция activeFilter() или middleware

3. Offset pagination на большой коллекции
   Skip 100000 → MongoDB сканирует и отбрасывает 100000 документов
   РЕШЕНИЕ: cursor-based pagination

4. Change stream без resume token
   При разрыве соединения — пропуск событий
   РЕШЕНИЕ: сохранять resume token после каждого успешного события

5. Тесты без replica set
   Standalone MongoDB не поддерживает транзакции и change streams
   РЕШЕНИЕ: testcontainers с WithReplicaSet

6. Один контейнер MongoDB на тест — медленно
   РЕШЕНИЕ: один контейнер в TestMain, разные database per test
```

---

## Вопросы на собеседовании

1. **Как реализовать soft delete в MongoDB?**
   Поле `deleted_at` вместо физического удаления. Все запросы фильтруют `deleted_at: {$exists: false}`. Unique-индексы — через partial filter expression. Периодический purge удаляет старые записи физически.

2. **Почему cursor-пагинация лучше offset?**
   Offset: `skip(N)` сканирует N документов — O(offset+limit). Cursor: `WHERE (created_at, _id) < (cursor)` с индексом — всегда O(limit). Cursor стабилен при вставке новых документов (нет дубликатов/пропусков).

3. **Что такое change streams и как обрабатывать разрывы соединения?**
   Change streams — подписка на изменения коллекции в реальном времени через oplog. При разрыве используется resume token для продолжения с точки останова. Token нужно сохранять в persistent storage после успешной обработки каждого события.

4. **Как тестировать MongoDB-код в Go?**
   testcontainers-go: запускает реальный MongoDB в Docker-контейнере. Один контейнер в TestMain, уникальная database per test. Для транзакций и change streams нужен replica set (`WithReplicaSet`).

5. **Зачем нужен optimistic locking и как его реализовать в MongoDB?**
   Защита от потерянных обновлений при конкурентном доступе. Поле `version` в документе, при update добавляем `version` в фильтр и `$inc: {version: 1}` в обновление. Если `matchedCount == 0` — кто-то обновил раньше, возвращаем конфликт.

6. **Как организовать создание индексов в production?**
   Индексы создаются при старте приложения через `CreateMany`. Операция идемпотентна — повторный вызов не создаёт дубликат. В production для больших коллекций использовать background build (в MongoDB 4.2+ все индексы строятся в background по умолчанию).
