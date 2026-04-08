# Singleton

## В Go

Go не имеет статических методов или классов. Singleton реализуется через `sync.Once` + package-level переменную.

```go
type Database struct {
    conn *sql.DB
}

var (
    instance *Database
    once     sync.Once
)

func GetDB() *Database {
    once.Do(func() {
        conn, err := sql.Open("postgres", dsn)
        if err != nil {
            log.Fatal(err)
        }
        instance = &Database{conn: conn}
    })
    return instance
}

// Потокобезопасно, ленивая инициализация
// once.Do гарантирует однократное выполнение
```

### Современный вариант (Go 1.21+)

```go
var getDB = sync.OnceValue(func() *Database {
    conn, _ := sql.Open("postgres", dsn)
    return &Database{conn: conn}
})

db := getDB() // вызов
```

### Когда использовать

- Пул соединений к БД
- Конфигурация приложения
- Логгер

### Когда НЕ использовать

- Затрудняет тестирование (глобальное состояние)
- Предпочитай явную передачу зависимостей (DI)
- В Go singleton часто заменяется на `init()` + package-level var или передачу через конструктор
