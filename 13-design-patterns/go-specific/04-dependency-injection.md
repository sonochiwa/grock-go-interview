# Dependency Injection

## Обзор

DI в Go — через интерфейсы в конструкторах. Нет фреймворков (wire, dig — опциональны). Просто передай зависимости явно.

```go
// Определяем интерфейс в пакете-потребителе
type UserRepository interface {
    GetByID(ctx context.Context, id int64) (*User, error)
    Create(ctx context.Context, user *User) error
}

type UserService struct {
    repo   UserRepository
    cache  Cache
    logger *slog.Logger
}

// Конструктор принимает интерфейсы
func NewUserService(repo UserRepository, cache Cache, logger *slog.Logger) *UserService {
    return &UserService{repo: repo, cache: cache, logger: logger}
}

// В main.go — wiring
func main() {
    db := postgres.NewDB(dsn)
    repo := postgres.NewUserRepo(db)
    cache := redis.NewCache(redisURL)
    logger := slog.Default()

    service := NewUserService(repo, cache, logger)
    handler := NewUserHandler(service)
    // ...
}
```

### Тестирование с моками

```go
type MockUserRepo struct {
    users map[int64]*User
}

func (m *MockUserRepo) GetByID(ctx context.Context, id int64) (*User, error) {
    u, ok := m.users[id]
    if !ok {
        return nil, ErrNotFound
    }
    return u, nil
}

func TestUserService(t *testing.T) {
    repo := &MockUserRepo{users: map[int64]*User{
        1: {ID: 1, Name: "Alice"},
    }}
    service := NewUserService(repo, &NoopCache{}, slog.Default())

    user, err := service.GetUser(context.Background(), 1)
    // ...
}
```

### Правила

1. **Принимай интерфейсы, возвращай структуры**
2. **Определяй интерфейс в потребителе**, не в провайдере
3. **Маленькие интерфейсы** (1-3 метода)
4. **Явный wiring в main()** — не магия, а просто конструкторы
5. **Не используй DI фреймворки** без необходимости — Go код должен быть явным
