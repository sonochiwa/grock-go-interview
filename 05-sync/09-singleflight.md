# singleflight

## Обзор

`golang.org/x/sync/singleflight` — дедупликация одновременных вызовов с одинаковым ключом. Идеально для предотвращения cache stampede.

## Концепции

```go
import "golang.org/x/sync/singleflight"

var group singleflight.Group

func GetUser(id int64) (*User, error) {
    key := fmt.Sprintf("user:%d", id)

    // Если 100 горутин вызовут GetUser(42) одновременно,
    // реальный запрос к БД выполнится ОДИН раз
    result, err, shared := group.Do(key, func() (any, error) {
        return db.QueryUser(id) // этот код выполнится один раз
    })
    // shared == true если результат был разделён между вызовами

    if err != nil {
        return nil, err
    }
    return result.(*User), nil
}
```

### Cache stampede prevention

```go
type Cache struct {
    mu    sync.RWMutex
    data  map[string]*User
    group singleflight.Group
}

func (c *Cache) GetUser(id int64) (*User, error) {
    key := fmt.Sprintf("user:%d", id)

    // Проверяем кеш
    c.mu.RLock()
    if user, ok := c.data[key]; ok {
        c.mu.RUnlock()
        return user, nil
    }
    c.mu.RUnlock()

    // Кеш промахнулся — singleflight предотвращает thundering herd
    result, err, _ := c.group.Do(key, func() (any, error) {
        user, err := db.QueryUser(id)
        if err != nil {
            return nil, err
        }
        c.mu.Lock()
        c.data[key] = user
        c.mu.Unlock()
        return user, nil
    })

    if err != nil {
        return nil, err
    }
    return result.(*User), nil
}
```

### Do vs DoChan

```go
// Do — блокирующий
result, err, shared := group.Do(key, fn)

// DoChan — неблокирующий, возвращает канал
ch := group.DoChan(key, fn)
select {
case result := <-ch:
    // result.Val, result.Err, result.Shared
case <-ctx.Done():
    return ctx.Err()
}
```

### Forget

```go
// Forget сбрасывает ожидающий вызов
// Полезно при кеш-инвалидации
group.Forget(key)
```

## Частые вопросы на собеседованиях

**Q: Что такое cache stampede?**
A: Когда кеш устаревает, сотни запросов одновременно идут в БД. Singleflight оставляет один запрос, остальные ждут результат.

**Q: Чем singleflight отличается от кеша?**
A: Singleflight дедуплицирует **текущие** вызовы, не хранит результат. Кеш хранит результат для будущих вызовов. Обычно используются вместе.
