# Mocks и Fakes

## Подходы

```
Mock: объект с запрограммированным поведением, проверяет вызовы
  + Проверяет "как" вызывается (аргументы, кол-во вызовов)
  - Хрупкие, привязаны к реализации

Fake: упрощённая рабочая реализация (in-memory DB)
  + Тестирует поведение, не привязан к реализации
  + Переиспользуется между тестами
  - Нужно поддерживать

Stub: возвращает фиксированные данные
  + Простейший
  - Не проверяет вызовы

Рекомендация Go community: hand-written fakes > generated mocks
```

## Hand-Written Fakes (рекомендуется)

```go
// Interface
type UserRepository interface {
    Get(ctx context.Context, id string) (*User, error)
    Create(ctx context.Context, user *User) error
    List(ctx context.Context, filter Filter) ([]*User, error)
}

// Fake
type fakeUserRepo struct {
    mu    sync.Mutex
    users map[string]*User
}

func newFakeUserRepo() *fakeUserRepo {
    return &fakeUserRepo{users: make(map[string]*User)}
}

func (r *fakeUserRepo) Get(_ context.Context, id string) (*User, error) {
    r.mu.Lock()
    defer r.mu.Unlock()
    user, ok := r.users[id]
    if !ok {
        return nil, ErrNotFound
    }
    return user, nil
}

func (r *fakeUserRepo) Create(_ context.Context, user *User) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    if _, exists := r.users[user.ID]; exists {
        return ErrAlreadyExists
    }
    r.users[user.ID] = user
    return nil
}

func (r *fakeUserRepo) List(_ context.Context, filter Filter) ([]*User, error) {
    r.mu.Lock()
    defer r.mu.Unlock()
    var result []*User
    for _, u := range r.users {
        result = append(result, u)
    }
    return result, nil
}

// Использование
func TestUserService(t *testing.T) {
    repo := newFakeUserRepo()
    svc := NewUserService(repo)

    err := svc.Register(context.Background(), "Alice", "alice@test.com")
    require.NoError(t, err)

    user, err := svc.GetByEmail(context.Background(), "alice@test.com")
    require.NoError(t, err)
    assert.Equal(t, "Alice", user.Name)
}
```

## gomock (generated mocks)

```go
//go:generate mockgen -source=repository.go -destination=mock_repository.go -package=service

// Использование
func TestUserServiceWithMock(t *testing.T) {
    ctrl := gomock.NewController(t)

    mockRepo := NewMockUserRepository(ctrl)

    // Ожидание: Get будет вызван с ID "123", вернёт пользователя
    mockRepo.EXPECT().
        Get(gomock.Any(), "123").
        Return(&User{ID: "123", Name: "Alice"}, nil).
        Times(1)

    svc := NewUserService(mockRepo)
    user, err := svc.GetUser(context.Background(), "123")

    require.NoError(t, err)
    assert.Equal(t, "Alice", user.Name)
}

// gomock matchers
mockRepo.EXPECT().
    Create(gomock.Any(), gomock.AssignableToTypeOf(&User{})).
    DoAndReturn(func(ctx context.Context, u *User) error {
        assert.NotEmpty(t, u.ID) // проверка внутри mock
        return nil
    })
```

## mockery (для testify)

```bash
# Генерация
mockery --name=UserRepository --output=mocks
```

```go
import "myapp/mocks"

func TestWithMockery(t *testing.T) {
    mockRepo := mocks.NewMockUserRepository(t) // auto cleanup

    mockRepo.EXPECT().
        Get(mock.Anything, "123").
        Return(&User{ID: "123", Name: "Alice"}, nil)

    svc := NewUserService(mockRepo)
    user, err := svc.GetUser(context.Background(), "123")

    require.NoError(t, err)
    assert.Equal(t, "Alice", user.Name)
}
```

## HTTP Mock

```go
// httptest.Server для mock внешних API
func TestExternalAPICall(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "/api/users/123", r.URL.Path)
        assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{
            "id":   "123",
            "name": "Alice",
        })
    }))
    defer server.Close()

    client := NewAPIClient(server.URL, "token123")
    user, err := client.GetUser(context.Background(), "123")

    require.NoError(t, err)
    assert.Equal(t, "Alice", user.Name)
}
```

## Когда что использовать

```
Hand-written fake:
  ✅ Repository/Store interfaces
  ✅ Логика в тестируемом коде, не в mock
  ✅ Переиспользование между тестами

Generated mock (gomock/mockery):
  ✅ Проверка конкретных вызовов (был вызван? с какими аргументами?)
  ✅ Внешние зависимости (API клиенты)
  ✅ Сложные сценарии (ошибки, таймауты)

httptest.Server:
  ✅ HTTP клиенты к внешним API

Реальные зависимости (testcontainers):
  ✅ Database repositories
  ✅ Redis/Kafka integration
  ✅ Финальная проверка перед production
```
