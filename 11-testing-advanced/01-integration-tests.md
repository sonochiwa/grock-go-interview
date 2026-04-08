# Integration Tests

## testcontainers-go

```go
import (
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
)

func setupPostgres(t *testing.T) *sql.DB {
    t.Helper()
    ctx := context.Background()

    pgContainer, err := postgres.Run(ctx,
        "postgres:16-alpine",
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
        postgres.WithInitScripts("testdata/schema.sql"), // миграции
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).WithStartupTimeout(30*time.Second),
        ),
    )
    require.NoError(t, err)

    t.Cleanup(func() {
        require.NoError(t, pgContainer.Terminate(ctx))
    })

    connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
    require.NoError(t, err)

    db, err := sql.Open("postgres", connStr)
    require.NoError(t, err)

    t.Cleanup(func() { db.Close() })
    return db
}

func TestUserRepository(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    db := setupPostgres(t)
    repo := NewUserRepository(db)

    t.Run("create and get", func(t *testing.T) {
        user := &User{Name: "Alice", Email: "alice@test.com"}
        err := repo.Create(context.Background(), user)
        require.NoError(t, err)
        assert.NotEmpty(t, user.ID)

        got, err := repo.GetByID(context.Background(), user.ID)
        require.NoError(t, err)
        assert.Equal(t, "Alice", got.Name)
    })
}
```

## Redis testcontainer

```go
import "github.com/testcontainers/testcontainers-go/modules/redis"

func setupRedis(t *testing.T) *redis.Client {
    t.Helper()
    ctx := context.Background()

    redisContainer, err := redis.Run(ctx, "redis:7-alpine")
    require.NoError(t, err)
    t.Cleanup(func() { redisContainer.Terminate(ctx) })

    endpoint, err := redisContainer.Endpoint(ctx, "")
    require.NoError(t, err)

    client := goredis.NewClient(&goredis.Options{Addr: endpoint})
    t.Cleanup(func() { client.Close() })
    return client
}
```

## TestMain (setup/teardown для пакета)

```go
var testDB *sql.DB

func TestMain(m *testing.M) {
    // Setup — один раз для всех тестов в пакете
    ctx := context.Background()
    pgContainer, err := postgres.Run(ctx, "postgres:16-alpine",
        postgres.WithDatabase("testdb"),
    )
    if err != nil {
        log.Fatal(err)
    }

    connStr, _ := pgContainer.ConnectionString(ctx, "sslmode=disable")
    testDB, _ = sql.Open("postgres", connStr)

    // Запуск тестов
    code := m.Run()

    // Teardown
    testDB.Close()
    pgContainer.Terminate(ctx)

    os.Exit(code)
}

func TestSomething(t *testing.T) {
    // Используем testDB
    repo := NewRepo(testDB)
    // ...
}
```

## Build Tags для integration тестов

```go
//go:build integration

package repo_test

// Этот файл компилируется только с: go test -tags integration
func TestIntegration(t *testing.T) {
    // ...
}
```

```bash
# Unit тесты (быстро)
go test ./...

# Integration тесты
go test -tags integration ./...

# Или через -short
go test ./...           # все тесты
go test -short ./...    # пропускает тесты с t.Skip в testing.Short()
```

## Test Fixtures и Cleanup

```go
func TestWithTransaction(t *testing.T) {
    db := setupPostgres(t)

    // Каждый subtest в своей транзакции → изоляция
    t.Run("test1", func(t *testing.T) {
        tx, err := db.BeginTx(context.Background(), nil)
        require.NoError(t, err)
        t.Cleanup(func() { tx.Rollback() }) // откат после теста

        repo := NewRepoWithTx(tx)
        // ... тест с чистой БД
    })

    t.Run("test2", func(t *testing.T) {
        tx, err := db.BeginTx(context.Background(), nil)
        require.NoError(t, err)
        t.Cleanup(func() { tx.Rollback() })

        repo := NewRepoWithTx(tx)
        // ... тоже чистая БД
    })
}
```

## HTTP Integration Test

```go
func TestAPIEndpoints(t *testing.T) {
    db := setupPostgres(t)
    app := NewApp(db) // создаёт router, handlers и т.д.

    srv := httptest.NewServer(app.Handler())
    defer srv.Close()

    t.Run("create user", func(t *testing.T) {
        body := `{"name":"Alice","email":"alice@test.com"}`
        resp, err := http.Post(srv.URL+"/api/v1/users", "application/json",
            strings.NewReader(body))
        require.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusCreated, resp.StatusCode)

        var user User
        json.NewDecoder(resp.Body).Decode(&user)
        assert.Equal(t, "Alice", user.Name)
        assert.NotEmpty(t, user.ID)
    })
}
```
