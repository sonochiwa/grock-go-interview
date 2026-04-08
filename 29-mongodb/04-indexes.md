# MongoDB: Индексы

## Зачем нужны индексы

Без индекса MongoDB выполняет **collection scan** — просматривает каждый документ. С индексом — обход B-tree структуры, O(log n).

```
Collection Scan (без индекса):     Index Scan:
┌───┐┌───┐┌───┐┌───┐┌───┐...     B-tree
│ 1 ││ 2 ││ 3 ││ 4 ││ 5 │        ┌───────┐
└───┘└───┘└───┘└───┘└───┘        │  50   │
  ↓    ↓    ↓    ↓    ↓          ┌┴───┐───┴┐
Проверяет КАЖДЫЙ документ       │ 25 │ 75 │
O(n)                            └┬─┬─┘─┬──┘
                                  ...  ...
                                O(log n)
```

## Типы индексов

### Single Field Index

```go
// Индекс по одному полю
indexModel := mongo.IndexModel{
    Keys: bson.D{{"email", 1}}, // 1 = ASC, -1 = DESC
}
name, err := coll.Indexes().CreateOne(ctx, indexModel)

// С опциями
indexModel := mongo.IndexModel{
    Keys: bson.D{{"email", 1}},
    Options: options.Index().
        SetUnique(true).                    // уникальный
        SetName("idx_email_unique").        // имя индекса
        SetBackground(true),               // не блокировать коллекцию (deprecated в 4.2+)
}
```

### Compound Index (составной)

```go
// Индекс по нескольким полям
// Порядок полей ВАЖЕН — определяет какие запросы используют индекс
indexModel := mongo.IndexModel{
    Keys: bson.D{
        {"status", 1},
        {"created_at", -1},
    },
}
name, err := coll.Indexes().CreateOne(ctx, indexModel)

// Этот индекс поддерживает запросы:
// 1. {status: "active"}                              — prefix match
// 2. {status: "active", created_at: {$gt: ...}}      — full match
// 3. {status: "active"} + sort({created_at: -1})     — index sort
//
// НЕ поддерживает:
// 4. {created_at: {$gt: ...}}                         — нет prefix
```

### ESR Rule (Equality, Sort, Range)

Порядок полей в составном индексе для оптимальной производительности:

```
1. Equality (=)  — поля с точным совпадением
2. Sort          — поля сортировки
3. Range         — поля с диапазоном ($gt, $lt, $in)

Пример запроса:
db.orders.find({
    status: "active",           // Equality
    total: { $gte: 100 }       // Range
}).sort({ created_at: -1 })    // Sort

Оптимальный индекс:
{ status: 1, created_at: -1, total: 1 }
  ↑ E          ↑ S              ↑ R
```

```go
// ESR index in Go
indexModel := mongo.IndexModel{
    Keys: bson.D{
        {"status", 1},      // Equality
        {"created_at", -1}, // Sort
        {"total", 1},       // Range
    },
}
```

### Multikey Index (для массивов)

MongoDB автоматически создаёт multikey index, если индексированное поле содержит массив:

```go
// Если документ: {tags: ["golang", "mongodb", "docker"]}
// Индекс по tags автоматически создаст записи для каждого элемента
indexModel := mongo.IndexModel{
    Keys: bson.D{{"tags", 1}},
}

// Теперь эффективно:
coll.Find(ctx, bson.M{"tags": "golang"})
coll.Find(ctx, bson.M{"tags": bson.M{"$in": bson.A{"golang", "rust"}}})

// ОГРАНИЧЕНИЕ: в составном индексе максимум ОДНО поле может быть массивом
// ПЛОХО: оба поля — массивы
// {tags: ["a", "b"], categories: ["x", "y"]}
// Индекс {tags: 1, categories: 1} — ОШИБКА при вставке
```

### Text Index

```go
// Полнотекстовый поиск
indexModel := mongo.IndexModel{
    Keys: bson.D{
        {"title", "text"},
        {"description", "text"},
    },
    Options: options.Index().
        SetDefaultLanguage("russian").
        SetWeights(bson.M{
            "title":       10, // title важнее
            "description": 5,
        }),
}
coll.Indexes().CreateOne(ctx, indexModel)

// Поиск
cursor, err := coll.Find(ctx, bson.M{
    "$text": bson.M{
        "$search": "golang mongodb",  // поиск по словам
    },
})

// Сортировка по релевантности
opts := options.Find().
    SetProjection(bson.M{"score": bson.M{"$meta": "textScore"}}).
    SetSort(bson.M{"score": bson.M{"$meta": "textScore"}})

// ОГРАНИЧЕНИЕ: только один text index на коллекцию
```

### TTL Index (Time-To-Live)

Автоматическое удаление документов через заданное время:

```go
// Удалять документы через 24 часа после created_at
indexModel := mongo.IndexModel{
    Keys:    bson.D{{"created_at", 1}},
    Options: options.Index().SetExpireAfterSeconds(86400), // 24 hours
}
coll.Indexes().CreateOne(ctx, indexModel)

// Удалять в конкретное время (expireAt поле содержит дату удаления)
indexModel := mongo.IndexModel{
    Keys:    bson.D{{"expire_at", 1}},
    Options: options.Index().SetExpireAfterSeconds(0), // удалить когда expire_at наступит
}

// При вставке указываем когда удалить
coll.InsertOne(ctx, bson.M{
    "session_id": "abc123",
    "expire_at":  time.Now().Add(30 * time.Minute), // удалить через 30 мин
})

// ВАЖНО: TTL background task запускается каждые 60 секунд
// Реальное удаление может произойти с задержкой до 60 сек
```

### Unique Index

```go
// Уникальный индекс
indexModel := mongo.IndexModel{
    Keys:    bson.D{{"email", 1}},
    Options: options.Index().SetUnique(true),
}

// Unique + Partial — уникальность только среди активных
indexModel := mongo.IndexModel{
    Keys: bson.D{{"email", 1}},
    Options: options.Index().
        SetUnique(true).
        SetPartialFilterExpression(bson.M{
            "deleted_at": bson.M{"$exists": false},
        }),
}

// Составной unique
indexModel := mongo.IndexModel{
    Keys: bson.D{
        {"user_id", 1},
        {"product_id", 1},
    },
    Options: options.Index().SetUnique(true), // уникальная пара
}
```

### Partial Index

Индексирует только документы, соответствующие фильтру:

```go
// Индекс только по активным пользователям
indexModel := mongo.IndexModel{
    Keys: bson.D{{"email", 1}},
    Options: options.Index().
        SetPartialFilterExpression(bson.M{
            "status": "active",
        }),
}

// Индекс меньше по размеру — быстрее обновляется и занимает меньше RAM
// Используется ТОЛЬКО если запрос включает условие из фильтра:
// ДА: {email: "alice@...", status: "active"}  — использует partial index
// НЕТ: {email: "alice@..."}                   — НЕ использует (статус не указан)
```

### Sparse Index

Индексирует только документы, где поле существует:

```go
indexModel := mongo.IndexModel{
    Keys: bson.D{{"phone", 1}},
    Options: options.Index().SetSparse(true),
}

// Документы без поля "phone" не попадут в индекс
// ВАЖНО: sparse index может давать неполные результаты при сортировке
// Если запрос использует sort по этому полю, документы без поля будут пропущены
```

### Hashed Index

Для hash-based sharding:

```go
indexModel := mongo.IndexModel{
    Keys: bson.D{{"user_id", "hashed"}},
}

// Поддерживает ТОЛЬКО equality ($eq), НЕ поддерживает range ($gt, $lt)
```

## Создание индексов в Go

### CreateOne / CreateMany

```go
func setupIndexes(ctx context.Context, coll *mongo.Collection) error {
    indexes := []mongo.IndexModel{
        {
            Keys:    bson.D{{"email", 1}},
            Options: options.Index().SetUnique(true).SetName("idx_email_unique"),
        },
        {
            Keys: bson.D{{"status", 1}, {"created_at", -1}},
            Options: options.Index().SetName("idx_status_created"),
        },
        {
            Keys:    bson.D{{"session_token", 1}},
            Options: options.Index().SetExpireAfterSeconds(3600).SetName("idx_session_ttl"),
        },
    }

    names, err := coll.Indexes().CreateMany(ctx, indexes)
    if err != nil {
        return fmt.Errorf("create indexes: %w", err)
    }

    for _, name := range names {
        log.Printf("created index: %s", name)
    }
    return nil
}
```

### Список индексов

```go
cursor, err := coll.Indexes().List(ctx)
if err != nil {
    return err
}
defer cursor.Close(ctx)

for cursor.Next(ctx) {
    var index bson.M
    cursor.Decode(&index)
    fmt.Printf("index: %v\n", index)
}
```

### Удаление индексов

```go
// Удалить один индекс по имени
_, err := coll.Indexes().DropOne(ctx, "idx_email_unique")

// Удалить все индексы (кроме _id)
_, err = coll.Indexes().DropAll(ctx)
```

## Explain

Анализ плана выполнения запроса. В Go используется через RunCommand:

```go
func explainQuery(ctx context.Context, db *mongo.Database, collName string, filter bson.M) (bson.M, error) {
    cmd := bson.D{
        {"explain", bson.D{
            {"find", collName},
            {"filter", filter},
        }},
        {"verbosity", "executionStats"},
    }

    var result bson.M
    err := db.RunCommand(ctx, cmd).Decode(&result)
    return result, err
}

// Использование
result, err := explainQuery(ctx, db, "users", bson.M{"email": "alice@example.com"})
```

### Ключевые поля explain

```
| Поле | Описание | Хорошо | Плохо |
|------|----------|--------|-------|
| winningPlan.stage | Тип операции | IXSCAN | COLLSCAN |
| totalDocsExamined | Просмотрено документов | ~= nReturned | >> nReturned |
| totalKeysExamined | Просмотрено ключей индекса | ~= nReturned | >> nReturned |
| nReturned | Возвращено документов | Ожидаемое кол-во | - |
| executionTimeMillis | Время выполнения | Малое | Большое |
```

```
Стадии плана (от лучшей к худшей):
├── IXSCAN → FETCH              — индекс + загрузка документов (нормально)
├── IXSCAN (covered query)      — только индекс, без загрузки документов (идеально)
├── COLLSCAN                     — полный скан коллекции (плохо)
└── SORT (in-memory)             — сортировка в памяти (плохо, лимит 100 MB)
```

## Covered Queries

Запрос, который полностью обслуживается индексом без обращения к документам:

```go
// Индекс: {email: 1, name: 1}

// Covered query — все поля запроса и проекции в индексе
opts := options.FindOne().SetProjection(bson.M{
    "email": 1,
    "name":  1,
    "_id":   0, // ВАЖНО: исключить _id, иначе нужен FETCH
})
coll.FindOne(ctx, bson.M{"email": "alice@example.com"}, opts)
// Plan: IXSCAN (без FETCH) — быстрее, не читает документы с диска

// НЕ covered — запрашивает поле age, которого нет в индексе
opts := options.FindOne().SetProjection(bson.M{
    "email": 1,
    "name":  1,
    "age":   1, // нет в индексе — нужен FETCH
    "_id":   0,
})
```

## Index Intersection

MongoDB может использовать несколько индексов для одного запроса:

```
Индекс 1: {status: 1}
Индекс 2: {city: 1}

Запрос: {status: "active", city: "Moscow"}
MongoDB МОЖЕТ использовать оба индекса и пересечь результаты.

НО: это обычно менее эффективно, чем составной индекс {status: 1, city: 1}.
Index intersection — fallback, а не оптимальная стратегия.
```

## Стратегии индексирования

### Правила

```
1. Индексируй поля, по которым фильтруешь и сортируешь
2. ESR rule для составных индексов: Equality → Sort → Range
3. Не создавай индексы "на всякий случай" — каждый индекс замедляет запись
4. Один составной индекс лучше нескольких одинарных
5. Используй partial/sparse для экономии размера
6. TTL индексы для автоочистки (сессии, логи, временные данные)
7. Мониторь неиспользуемые индексы через $indexStats
```

### Мониторинг использования индексов

```go
// $indexStats — статистика использования каждого индекса
pipeline := bson.A{
    bson.M{"$indexStats": bson.M{}},
}

cursor, err := coll.Aggregate(ctx, pipeline)
if err != nil {
    return err
}
defer cursor.Close(ctx)

for cursor.Next(ctx) {
    var stat bson.M
    cursor.Decode(&stat)
    fmt.Printf("index: %v, accesses: %v\n", stat["name"], stat["accesses"])
}
// Индексы с accesses.ops = 0 — кандидаты на удаление
```

### Размер индексов

```go
// Статистика коллекции включая размер индексов
cmd := bson.D{{"collStats", "users"}}
var stats bson.M
db.RunCommand(ctx, cmd).Decode(&stats)

fmt.Printf("data size: %v\n", stats["size"])
fmt.Printf("index size: %v\n", stats["totalIndexSize"])
fmt.Printf("indexes: %v\n", stats["indexSizes"])

// Все индексы должны помещаться в RAM для оптимальной производительности
// Если totalIndexSize > доступной RAM, производительность деградирует
```

## Типичные ошибки

```
1. COLLSCAN на production — нет индекса для частого запроса
   РЕШЕНИЕ: анализировать slow query log, добавить индексы

2. Слишком много индексов — замедляет запись
   Каждый INSERT обновляет ВСЕ индексы коллекции
   РЕШЕНИЕ: удалить неиспользуемые ($indexStats), объединить в составные

3. Неправильный порядок полей в составном индексе
   {created_at: 1, status: 1} для запроса {status: "active"} — бесполезен
   РЕШЕНИЕ: ESR rule, equality поля первыми

4. Sort in memory > 100 MB — ошибка
   Без подходящего индекса сортировка выполняется в памяти
   Лимит 100 MB, при превышении — ошибка
   РЕШЕНИЕ: индекс должен покрывать сортировку

5. Забыли исключить _id в covered query
   _id включается по умолчанию, нужен FETCH для загрузки
   РЕШЕНИЕ: явно {_id: 0} в проекции
```

---

## Вопросы на собеседовании

1. **Какие типы индексов есть в MongoDB?**
   Single field, compound, multikey (массивы), text (полнотекстовый), hashed (для sharding), TTL (автоудаление), unique, partial/sparse, wildcard, 2dsphere (геолокация).

2. **Что такое ESR rule?**
   Порядок полей в составном индексе: Equality (точное совпадение) первыми, затем Sort (поля сортировки), затем Range (диапазоны $gt/$lt). Такой порядок минимизирует количество ключей индекса, которые нужно просканировать.

3. **Что такое covered query?**
   Запрос, который полностью обслуживается индексом без чтения документов. Все поля фильтра и проекции должны быть в индексе. Поле `_id` нужно явно исключить (`_id: 0`). В explain — IXSCAN без FETCH.

4. **Как TTL index работает?**
   MongoDB background thread каждые 60 секунд проверяет документы и удаляет те, чья дата в индексированном поле + `expireAfterSeconds` < текущее время. Удаление может задерживаться до 60 секунд. TTL работает только на полях с типом Date.

5. **Чем partial index отличается от sparse?**
   Partial index использует произвольное условие (`partialFilterExpression`) — более гибкий. Sparse index просто исключает документы, где поле не существует. Partial может фильтровать по значению, типу, нескольким условиям. Sparse — только по наличию поля.

6. **Как узнать, используется ли индекс в запросе?**
   `explain("executionStats")` — показывает план выполнения. `IXSCAN` = используется индекс, `COLLSCAN` = полный скан. `$indexStats` — показывает статистику использования всех индексов коллекции.

7. **Почему много индексов — это плохо?**
   Каждая запись (insert/update/delete) обновляет все затронутые индексы. Больше индексов = медленнее запись. Индексы занимают RAM — если не помещаются, происходит page faulting и деградация чтения. Нужен баланс между скоростью чтения и записи.
