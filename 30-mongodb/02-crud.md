# MongoDB: CRUD операции

## BSON-типы

MongoDB хранит данные в формате BSON (Binary JSON). Go-драйвер предоставляет несколько типов для работы с BSON:

| Тип | Описание | Сохраняет порядок | Пример |
|-----|----------|-------------------|--------|
| `bson.M` | Map (`map[string]interface{}`) | Нет | `bson.M{"name": "Alice", "age": 30}` |
| `bson.D` | Ordered document (slice of E) | Да | `bson.D{{"name", "Alice"}, {"age", 30}}` |
| `bson.A` | Array (slice of interface{}) | Да | `bson.A{"red", "green", "blue"}` |
| `bson.E` | Single element (key-value pair) | - | `bson.E{Key: "name", Value: "Alice"}` |

### Когда что использовать

```go
// bson.M — для фильтров и простых документов (порядок не важен)
filter := bson.M{"status": "active", "age": bson.M{"$gte": 18}}

// bson.D — когда порядок ключей важен (команды, индексы, сортировка)
sort := bson.D{{"created_at", -1}, {"name", 1}}

// bson.A — для массивов в запросах
filter := bson.M{"status": bson.M{"$in": bson.A{"active", "pending"}}}

// На собеседовании: bson.D нужен для команд, где порядок ключей
// имеет значение (например, составные индексы). bson.M проще,
// но Go map не гарантирует порядок итерации.
```

### Маппинг Go-типов в BSON

```
| Go тип | BSON тип |
|--------|----------|
| string | String |
| int, int32 | Int32 |
| int64 | Int64 |
| float64 | Double |
| bool | Boolean |
| time.Time | DateTime |
| []byte | Binary |
| primitive.ObjectID | ObjectID |
| nil | Null |
| bson.M / struct | Document |
| slice / bson.A | Array |
```

## InsertOne

```go
type User struct {
    ID        primitive.ObjectID `bson:"_id,omitempty"`
    Name      string             `bson:"name"`
    Email     string             `bson:"email"`
    Age       int                `bson:"age"`
    Tags      []string           `bson:"tags,omitempty"`
    CreatedAt time.Time          `bson:"created_at"`
}

func (r *UserRepo) Create(ctx context.Context, user *User) error {
    user.CreatedAt = time.Now()

    result, err := r.coll.InsertOne(ctx, user)
    if err != nil {
        // Check for duplicate key error
        if mongo.IsDuplicateKeyError(err) {
            return ErrAlreadyExists
        }
        return fmt.Errorf("insert user: %w", err)
    }

    // result.InsertedID contains the generated _id
    user.ID = result.InsertedID.(primitive.ObjectID)
    return nil
}
```

## InsertMany

```go
func (r *UserRepo) CreateMany(ctx context.Context, users []User) ([]primitive.ObjectID, error) {
    // Convert to []interface{}
    docs := make([]interface{}, len(users))
    for i := range users {
        users[i].CreatedAt = time.Now()
        docs[i] = users[i]
    }

    result, err := r.coll.InsertMany(ctx, docs)
    if err != nil {
        return nil, fmt.Errorf("insert many: %w", err)
    }

    ids := make([]primitive.ObjectID, len(result.InsertedIDs))
    for i, id := range result.InsertedIDs {
        ids[i] = id.(primitive.ObjectID)
    }
    return ids, nil
}

// Ordered vs Unordered inserts
opts := options.InsertMany().SetOrdered(false)
// ordered=true (default): останавливается при первой ошибке
// ordered=false: продолжает вставку остальных при ошибке — быстрее для bulk
result, err := coll.InsertMany(ctx, docs, opts)
```

## FindOne

```go
func (r *UserRepo) GetByID(ctx context.Context, id primitive.ObjectID) (User, error) {
    var user User
    err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
    if err != nil {
        if errors.Is(err, mongo.ErrNoDocuments) {
            return User{}, ErrNotFound
        }
        return User{}, fmt.Errorf("find user: %w", err)
    }
    return user, nil
}

// С проекцией — выбираем только нужные поля
func (r *UserRepo) GetEmail(ctx context.Context, id primitive.ObjectID) (string, error) {
    opts := options.FindOne().SetProjection(bson.M{
        "email": 1, // 1 = include
        "_id":   0, // 0 = exclude
    })

    var result struct {
        Email string `bson:"email"`
    }
    err := r.coll.FindOne(ctx, bson.M{"_id": id}, opts).Decode(&result)
    if errors.Is(err, mongo.ErrNoDocuments) {
        return "", ErrNotFound
    }
    return result.Email, err
}
```

## Find (несколько документов)

```go
func (r *UserRepo) ListActive(ctx context.Context, limit, skip int64) ([]User, error) {
    filter := bson.M{"status": "active"}

    opts := options.Find().
        SetSort(bson.D{{"created_at", -1}}).   // сортировка: -1 = DESC, 1 = ASC
        SetLimit(limit).                         // максимум документов
        SetSkip(skip).                           // пропустить N документов
        SetProjection(bson.M{                    // выбрать поля
            "name":       1,
            "email":      1,
            "created_at": 1,
        })

    cursor, err := r.coll.Find(ctx, filter, opts)
    if err != nil {
        return nil, fmt.Errorf("find users: %w", err)
    }
    defer cursor.Close(ctx)

    var users []User
    if err := cursor.All(ctx, &users); err != nil {
        return nil, fmt.Errorf("decode users: %w", err)
    }
    return users, nil
}
```

### Итерация по курсору

```go
// cursor.All() загружает ВСЕ документы в память
// Для больших результатов используй итерацию:

cursor, err := coll.Find(ctx, filter)
if err != nil {
    return err
}
defer cursor.Close(ctx) // ОБЯЗАТЕЛЬНО закрыть курсор

for cursor.Next(ctx) {
    var user User
    if err := cursor.Decode(&user); err != nil {
        return fmt.Errorf("decode: %w", err)
    }
    // process user...
}

// Проверка ошибок после итерации (аналогично rows.Err() в SQL)
if err := cursor.Err(); err != nil {
    return fmt.Errorf("cursor error: %w", err)
}
```

## Фильтры

### Операторы сравнения

```go
// Равенство (неявное)
bson.M{"status": "active"}

// $eq — явное равенство
bson.M{"age": bson.M{"$eq": 25}}

// $ne — не равно
bson.M{"status": bson.M{"$ne": "deleted"}}

// $gt, $gte, $lt, $lte — больше/меньше
bson.M{"age": bson.M{"$gte": 18, "$lte": 65}}

// $in — значение в списке
bson.M{"status": bson.M{"$in": bson.A{"active", "pending"}}}

// $nin — значение НЕ в списке
bson.M{"role": bson.M{"$nin": bson.A{"admin", "superadmin"}}}
```

### Логические операторы

```go
// $and — неявный (несколько условий в одном bson.M)
bson.M{"status": "active", "age": bson.M{"$gte": 18}}

// $and — явный (для нескольких условий на одно поле)
bson.M{"$and": bson.A{
    bson.M{"price": bson.M{"$gte": 10}},
    bson.M{"price": bson.M{"$lte": 100}},
}}

// $or
bson.M{"$or": bson.A{
    bson.M{"status": "active"},
    bson.M{"role": "admin"},
}}

// $not
bson.M{"age": bson.M{"$not": bson.M{"$lt": 18}}}

// $nor — ни одно из условий не выполнено
bson.M{"$nor": bson.A{
    bson.M{"status": "deleted"},
    bson.M{"banned": true},
}}
```

### Операторы элементов

```go
// $exists — поле существует/не существует
bson.M{"deleted_at": bson.M{"$exists": false}}

// $type — тип поля
bson.M{"age": bson.M{"$type": "int"}}
```

### Операторы массивов

```go
// Элемент есть в массиве
bson.M{"tags": "golang"} // tags содержит "golang"

// $all — массив содержит ВСЕ элементы
bson.M{"tags": bson.M{"$all": bson.A{"golang", "mongodb"}}}

// $size — длина массива
bson.M{"tags": bson.M{"$size": 3}}

// $elemMatch — элемент массива соответствует нескольким условиям
bson.M{"scores": bson.M{"$elemMatch": bson.M{
    "$gte": 80,
    "$lte": 100,
}}}
```

### Регулярные выражения

```go
// $regex — поиск по регулярному выражению
bson.M{"name": bson.M{"$regex": "^Alice", "$options": "i"}} // case-insensitive

// Через primitive.Regex
bson.M{"email": primitive.Regex{Pattern: `@gmail\.com$`, Options: "i"}}
```

## UpdateOne

```go
func (r *UserRepo) UpdateName(ctx context.Context, id primitive.ObjectID, name string) error {
    filter := bson.M{"_id": id}
    update := bson.M{
        "$set": bson.M{
            "name":       name,
            "updated_at": time.Now(),
        },
    }

    result, err := r.coll.UpdateOne(ctx, filter, update)
    if err != nil {
        return fmt.Errorf("update user: %w", err)
    }
    if result.MatchedCount == 0 {
        return ErrNotFound
    }
    return nil
}
```

### Операторы обновления

```go
// $set — установить значение поля
bson.M{"$set": bson.M{"name": "Bob", "age": 31}}

// $unset — удалить поле из документа
bson.M{"$unset": bson.M{"temporary_field": ""}}

// $inc — увеличить числовое значение
bson.M{"$inc": bson.M{"views": 1, "score": -5}}

// $mul — умножить числовое значение
bson.M{"$mul": bson.M{"price": 1.1}} // увеличить на 10%

// $min / $max — обновить только если новое значение меньше/больше текущего
bson.M{"$min": bson.M{"low_score": 50}}  // обновит, только если 50 < текущего
bson.M{"$max": bson.M{"high_score": 95}} // обновит, только если 95 > текущего

// $rename — переименовать поле
bson.M{"$rename": bson.M{"old_name": "new_name"}}

// $currentDate — установить текущую дату
bson.M{"$currentDate": bson.M{"updated_at": true}}
```

### Операторы массивов (update)

```go
// $push — добавить элемент в массив
bson.M{"$push": bson.M{"tags": "new-tag"}}

// $push с $each — добавить несколько элементов
bson.M{"$push": bson.M{
    "tags": bson.M{"$each": bson.A{"tag1", "tag2", "tag3"}},
}}

// $push с $each + $sort + $slice — поддерживать топ-N
bson.M{"$push": bson.M{
    "scores": bson.M{
        "$each":  bson.A{95},
        "$sort":  -1,     // сортировать по убыванию
        "$slice": 10,     // хранить только топ 10
    },
}}

// $pull — удалить элемент из массива
bson.M{"$pull": bson.M{"tags": "old-tag"}}

// $pull с условием
bson.M{"$pull": bson.M{"scores": bson.M{"$lt": 50}}}

// $addToSet — добавить в массив, если ещё нет (аналог Set)
bson.M{"$addToSet": bson.M{"tags": "unique-tag"}}

// $pop — удалить первый (-1) или последний (1) элемент
bson.M{"$pop": bson.M{"queue": -1}} // удалить первый

// $ — позиционный оператор (обновить первый совпавший элемент массива)
// Обновить рейтинг пользователя с user_id = 42 в массиве ratings
filter := bson.M{"ratings.user_id": 42}
update := bson.M{"$set": bson.M{"ratings.$.score": 5}}
```

## UpdateMany

```go
func (r *UserRepo) DeactivateOld(ctx context.Context, before time.Time) (int64, error) {
    filter := bson.M{
        "last_login": bson.M{"$lt": before},
        "status":     "active",
    }
    update := bson.M{
        "$set": bson.M{
            "status":     "inactive",
            "updated_at": time.Now(),
        },
    }

    result, err := r.coll.UpdateMany(ctx, filter, update)
    if err != nil {
        return 0, fmt.Errorf("update many: %w", err)
    }
    return result.ModifiedCount, nil
}
```

## ReplaceOne

Полная замена документа (все поля, кроме `_id`):

```go
func (r *UserRepo) Replace(ctx context.Context, user User) error {
    filter := bson.M{"_id": user.ID}

    result, err := r.coll.ReplaceOne(ctx, filter, user)
    if err != nil {
        return fmt.Errorf("replace user: %w", err)
    }
    if result.MatchedCount == 0 {
        return ErrNotFound
    }
    return nil
}

// ReplaceOne vs UpdateOne:
// ReplaceOne — полностью заменяет документ (кроме _id)
// UpdateOne — частичное обновление через операторы ($set, $inc, etc.)
```

## DeleteOne / DeleteMany

```go
func (r *UserRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
    result, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
    if err != nil {
        return fmt.Errorf("delete user: %w", err)
    }
    if result.DeletedCount == 0 {
        return ErrNotFound
    }
    return nil
}

func (r *UserRepo) DeleteInactive(ctx context.Context, before time.Time) (int64, error) {
    filter := bson.M{
        "status":     "inactive",
        "updated_at": bson.M{"$lt": before},
    }

    result, err := r.coll.DeleteMany(ctx, filter)
    if err != nil {
        return 0, fmt.Errorf("delete many: %w", err)
    }
    return result.DeletedCount, nil
}
```

## FindOneAndUpdate / FindOneAndDelete

Атомарные операции — находят документ и изменяют его за одну операцию. Полезны для конкурентного доступа.

```go
// FindOneAndUpdate — найти и обновить, вернуть документ
func (r *JobRepo) TakeJob(ctx context.Context, workerID string) (*Job, error) {
    filter := bson.M{"status": "pending"}
    update := bson.M{
        "$set": bson.M{
            "status":    "processing",
            "worker_id": workerID,
            "taken_at":  time.Now(),
        },
    }

    // ReturnDocument: After — вернуть документ ПОСЛЕ обновления
    // ReturnDocument: Before (default) — вернуть документ ДО обновления
    opts := options.FindOneAndUpdate().
        SetReturnDocument(options.After).
        SetSort(bson.D{{"priority", -1}, {"created_at", 1}})

    var job Job
    err := r.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&job)
    if errors.Is(err, mongo.ErrNoDocuments) {
        return nil, nil // no pending jobs
    }
    if err != nil {
        return nil, fmt.Errorf("take job: %w", err)
    }
    return &job, nil
}

// FindOneAndDelete — найти и удалить, вернуть удалённый документ
func (r *QueueRepo) Pop(ctx context.Context) (*Message, error) {
    opts := options.FindOneAndDelete().
        SetSort(bson.D{{"created_at", 1}}) // FIFO

    var msg Message
    err := r.coll.FindOneAndDelete(ctx, bson.M{}, opts).Decode(&msg)
    if errors.Is(err, mongo.ErrNoDocuments) {
        return nil, nil
    }
    return &msg, err
}
```

## Upsert

Insert если документ не найден, Update если найден:

```go
func (r *StatsRepo) IncrementViews(ctx context.Context, pageURL string) error {
    filter := bson.M{"url": pageURL}
    update := bson.M{
        "$inc": bson.M{"views": 1},
        "$setOnInsert": bson.M{ // поля только при insert
            "url":        pageURL,
            "created_at": time.Now(),
        },
    }

    opts := options.Update().SetUpsert(true)
    _, err := r.coll.UpdateOne(ctx, filter, update, opts)
    return err
}
```

## BulkWrite

Несколько разных операций за один round-trip:

```go
func (r *UserRepo) BulkOps(ctx context.Context) error {
    models := []mongo.WriteModel{
        mongo.NewInsertOneModel().SetDocument(bson.M{
            "name": "Alice", "status": "active",
        }),
        mongo.NewUpdateOneModel().
            SetFilter(bson.M{"name": "Bob"}).
            SetUpdate(bson.M{"$set": bson.M{"status": "active"}}),
        mongo.NewDeleteOneModel().
            SetFilter(bson.M{"status": "deleted"}),
    }

    opts := options.BulkWrite().SetOrdered(false) // parallel execution
    result, err := r.coll.BulkWrite(ctx, models, opts)
    if err != nil {
        return fmt.Errorf("bulk write: %w", err)
    }

    fmt.Printf("inserted: %d, updated: %d, deleted: %d\n",
        result.InsertedCount, result.ModifiedCount, result.DeletedCount)
    return nil
}
```

## CountDocuments / EstimatedDocumentCount

```go
// Точный подсчёт (выполняет aggregation pipeline)
count, err := coll.CountDocuments(ctx, bson.M{"status": "active"})

// Приблизительный подсчёт (из метаданных коллекции — O(1))
// Не принимает фильтр! Возвращает общее количество документов
total, err := coll.EstimatedDocumentCount(ctx)
```

## Distinct

```go
// Уникальные значения поля
values, err := coll.Distinct(ctx, "status", bson.M{})
// values: ["active", "inactive", "pending"]

// С фильтром
cities, err := coll.Distinct(ctx, "city", bson.M{"country": "Russia"})
```

## Типичные ошибки

```go
// 1. Забыли закрыть курсор — утечка ресурсов
cursor, _ := coll.Find(ctx, filter)
for cursor.Next(ctx) { /* ... */ }
// cursor.Close(ctx) не вызван!

// 2. Не проверили cursor.Err() — пропущена ошибка сети
cursor, _ := coll.Find(ctx, filter)
defer cursor.Close(ctx)
for cursor.Next(ctx) { /* ... */ }
// cursor.Err() не проверен — сетевая ошибка молча проглочена

// 3. UpdateOne без оператора ($set) — ОШИБКА!
coll.UpdateOne(ctx,
    bson.M{"_id": id},
    bson.M{"name": "Alice"}, // ПЛОХО! Нужен $set
)
// Правильно:
coll.UpdateOne(ctx,
    bson.M{"_id": id},
    bson.M{"$set": bson.M{"name": "Alice"}},
)

// 4. cursor.All() на огромную коллекцию — OOM
cursor, _ := coll.Find(ctx, bson.M{}) // миллионы документов
var all []User
cursor.All(ctx, &all) // загрузит ВСЁ в память
// Правильно: итерация через cursor.Next() или использовать limit
```

---

## Вопросы на собеседовании

1. **В чём разница между `bson.M` и `bson.D`?**
   `bson.M` — map, не гарантирует порядок ключей, удобнее для фильтров. `bson.D` — ordered slice of key-value pairs, нужен когда порядок ключей важен (составные индексы, команды, сортировка).

2. **Чем `UpdateOne` отличается от `ReplaceOne`?**
   `UpdateOne` использует операторы (`$set`, `$inc`) для частичного обновления. `ReplaceOne` полностью заменяет документ (кроме `_id`). `UpdateOne` с `$set` обновляет только указанные поля, `ReplaceOne` удалит все поля, которых нет в новом документе.

3. **Что такое `FindOneAndUpdate` и когда его использовать?**
   Атомарная операция: найти + обновить за один round-trip. Полезна для конкурентного доступа (например, взять задачу из очереди). Гарантирует, что два воркера не возьмут одну задачу. Аналог `SELECT FOR UPDATE` в SQL.

4. **Зачем нужен `$setOnInsert` в upsert?**
   `$setOnInsert` устанавливает поля только при создании нового документа (insert часть upsert). Не затрагивает существующий документ при update. Полезно для полей вроде `created_at`.

5. **Чем `CountDocuments` отличается от `EstimatedDocumentCount`?**
   `CountDocuments` выполняет aggregation pipeline с фильтром — точный результат, но может быть медленным на больших коллекциях. `EstimatedDocumentCount` использует метаданные коллекции — O(1), но не принимает фильтр и может быть неточным после нештатных ситуаций.

6. **Что произойдёт при `InsertMany` с `ordered=false` если одна вставка упадёт?**
   При `ordered=false` драйвер продолжит вставку остальных документов. Ошибка будет содержать информацию о неудачных вставках (BulkWriteException). При `ordered=true` (default) вставка остановится на первой ошибке.
