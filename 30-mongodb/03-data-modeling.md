# MongoDB: Моделирование данных

## Embedding vs Referencing

Главное решение при проектировании схемы MongoDB — встраивать связанные данные в один документ (embedding) или хранить отдельно со ссылками (referencing).

### Embedding (встраивание)

```go
// Адрес встроен в документ пользователя
type User struct {
    ID      primitive.ObjectID `bson:"_id,omitempty"`
    Name    string             `bson:"name"`
    Email   string             `bson:"email"`
    Address Address            `bson:"address"` // embedded document
    Orders  []Order            `bson:"orders"`  // embedded array
}

type Address struct {
    Street string `bson:"street"`
    City   string `bson:"city"`
    ZIP    string `bson:"zip"`
}

type Order struct {
    ID        primitive.ObjectID `bson:"_id,omitempty"`
    Product   string             `bson:"product"`
    Amount    float64            `bson:"amount"`
    CreatedAt time.Time          `bson:"created_at"`
}
```

BSON-документ в MongoDB:

```json
{
    "_id": ObjectId("..."),
    "name": "Alice",
    "email": "alice@example.com",
    "address": {
        "street": "Main St 42",
        "city": "Moscow",
        "zip": "101000"
    },
    "orders": [
        {"product": "Book", "amount": 15.99, "created_at": ISODate("...")},
        {"product": "Pen", "amount": 3.50, "created_at": ISODate("...")}
    ]
}
```

### Referencing (ссылки)

```go
// Заказы хранятся отдельно, ссылаются на пользователя через user_id
type User struct {
    ID    primitive.ObjectID `bson:"_id,omitempty"`
    Name  string             `bson:"name"`
    Email string             `bson:"email"`
}

type Order struct {
    ID        primitive.ObjectID `bson:"_id,omitempty"`
    UserID    primitive.ObjectID `bson:"user_id"` // reference to users collection
    Product   string             `bson:"product"`
    Amount    float64            `bson:"amount"`
    CreatedAt time.Time          `bson:"created_at"`
}
```

### Сравнение подходов

```
| Критерий | Embedding | Referencing |
|----------|-----------|-------------|
| Чтение | Один запрос (быстро) | Два запроса или $lookup (медленнее) |
| Запись | Атомарный update | Два update (нужна транзакция) |
| Размер документа | Растёт (лимит 16 MB) | Фиксированный |
| Дублирование | Данные могут дублироваться | Нет дублирования |
| Связь 1:1, 1:few | Идеально | Избыточно |
| Связь 1:many (тысячи) | Документ раздувается | Правильный выбор |
| Связь many:many | Не подходит | Правильный выбор |
| Обновление вложенных | Сложнее (позиционные операторы) | Простой update по _id |
```

### Правило принятия решения

```
Embedding когда:
├── Связь 1:1 или 1:few (< 100 элементов)
├── Данные всегда читаются вместе
├── Вложенные данные не нужны сами по себе
└── Вложенный массив не растёт бесконечно

Referencing когда:
├── Связь 1:many (тысячи) или many:many
├── Данные читаются независимо
├── Вложенные данные часто обновляются
├── Размер документа приближается к 16 MB
└── Данные нужны в нескольких коллекциях
```

## Schema Design Patterns

### Subset Pattern

Проблема: документ содержит массив с тысячами элементов, но обычно нужны только последние N.

```go
// ПЛОХО: все отзывы встроены в документ продукта
type Product struct {
    ID      primitive.ObjectID `bson:"_id,omitempty"`
    Name    string             `bson:"name"`
    Reviews []Review           `bson:"reviews"` // может быть 10000+ отзывов
}

// ХОРОШО: Subset Pattern — последние N отзывов в документе, остальные отдельно
type Product struct {
    ID            primitive.ObjectID `bson:"_id,omitempty"`
    Name          string             `bson:"name"`
    RecentReviews []Review           `bson:"recent_reviews"` // последние 10
    ReviewCount   int                `bson:"review_count"`
}

// Полная коллекция отзывов — для страницы "все отзывы"
type Review struct {
    ID        primitive.ObjectID `bson:"_id,omitempty"`
    ProductID primitive.ObjectID `bson:"product_id"`
    UserID    primitive.ObjectID `bson:"user_id"`
    Rating    int                `bson:"rating"`
    Text      string             `bson:"text"`
    CreatedAt time.Time          `bson:"created_at"`
}

// При добавлении отзыва — обновляем и коллекцию reviews, и subset в product
func (r *ReviewRepo) Add(ctx context.Context, review Review) error {
    // 1. Insert в коллекцию reviews
    _, err := r.reviews.InsertOne(ctx, review)
    if err != nil {
        return err
    }

    // 2. Push в recent_reviews с ограничением в 10 ($slice)
    _, err = r.products.UpdateOne(ctx,
        bson.M{"_id": review.ProductID},
        bson.M{
            "$push": bson.M{
                "recent_reviews": bson.M{
                    "$each":  bson.A{review},
                    "$sort":  bson.M{"created_at": -1},
                    "$slice": 10,
                },
            },
            "$inc": bson.M{"review_count": 1},
        },
    )
    return err
}
```

### Bucket Pattern

Проблема: большое количество мелких документов (time-series, логи, IoT).

```go
// ПЛОХО: один документ на каждое измерение
// { "sensor_id": 1, "temp": 22.5, "ts": ISODate("2025-01-15T10:00:00Z") }
// Миллионы мелких документов = большой overhead на _id, индексы, метаданные

// ХОРОШО: группировка по временным интервалам (bucket)
type SensorBucket struct {
    ID        primitive.ObjectID `bson:"_id,omitempty"`
    SensorID  string             `bson:"sensor_id"`
    StartDate time.Time          `bson:"start_date"` // начало часа
    EndDate   time.Time          `bson:"end_date"`   // конец часа
    Count     int                `bson:"count"`
    Sum       float64            `bson:"sum"`         // для быстрого AVG
    Readings  []Reading          `bson:"readings"`
}

type Reading struct {
    Temp      float64   `bson:"temp"`
    Timestamp time.Time `bson:"ts"`
}

// Добавление показания в bucket
func (r *SensorRepo) AddReading(ctx context.Context, sensorID string, temp float64, ts time.Time) error {
    bucketStart := ts.Truncate(time.Hour)
    bucketEnd := bucketStart.Add(time.Hour)

    filter := bson.M{
        "sensor_id":  sensorID,
        "start_date": bucketStart,
        "count":      bson.M{"$lt": 200}, // max 200 readings per bucket
    }
    update := bson.M{
        "$push": bson.M{"readings": Reading{Temp: temp, Timestamp: ts}},
        "$inc":  bson.M{"count": 1, "sum": temp},
        "$setOnInsert": bson.M{
            "sensor_id":  sensorID,
            "start_date": bucketStart,
            "end_date":   bucketEnd,
        },
    }
    opts := options.Update().SetUpsert(true)
    _, err := r.coll.UpdateOne(ctx, filter, update, opts)
    return err
}
```

Преимущества bucket pattern: меньше документов (меньше overhead), агрегации быстрее (меньше документов сканировать), предвычисленные sum/count ускоряют аналитику.

### Outlier Pattern

Проблема: 99% документов имеют небольшой массив, но 1% — огромный (например, у звёзд миллионы подписчиков).

```go
// Обычный пользователь — подписчики встроены
type User struct {
    ID          primitive.ObjectID   `bson:"_id,omitempty"`
    Name        string               `bson:"name"`
    Followers   []primitive.ObjectID `bson:"followers"`
    HasOverflow bool                 `bson:"has_overflow"` // flag for outliers
}

// Overflow-документы для пользователей с > 1000 подписчиков
type FollowerOverflow struct {
    ID        primitive.ObjectID   `bson:"_id,omitempty"`
    UserID    primitive.ObjectID   `bson:"user_id"`
    Page      int                  `bson:"page"`
    Followers []primitive.ObjectID `bson:"followers"`
}

func (r *UserRepo) AddFollower(ctx context.Context, userID, followerID primitive.ObjectID) error {
    // Try to push into the main document (limit 1000)
    result, err := r.users.UpdateOne(ctx,
        bson.M{
            "_id":       userID,
            "followers": bson.M{"$not": bson.M{"$size": 1000}},
        },
        bson.M{"$addToSet": bson.M{"followers": followerID}},
    )
    if err != nil {
        return err
    }

    if result.MatchedCount == 0 {
        // Main document is full — use overflow
        r.users.UpdateOne(ctx,
            bson.M{"_id": userID},
            bson.M{"$set": bson.M{"has_overflow": true}},
        )
        // Upsert into overflow collection
        _, err = r.overflow.UpdateOne(ctx,
            bson.M{
                "user_id":   userID,
                "followers": bson.M{"$not": bson.M{"$size": 1000}},
            },
            bson.M{"$addToSet": bson.M{"followers": followerID}},
            options.Update().SetUpsert(true),
        )
        return err
    }
    return nil
}
```

### Computed Pattern

Проблема: часто запрашиваемые агрегации (count, sum, avg) медленно вычислять каждый раз.

```go
// Предвычисленные значения обновляются при каждой записи
type Product struct {
    ID          primitive.ObjectID `bson:"_id,omitempty"`
    Name        string             `bson:"name"`
    Price       float64            `bson:"price"`
    // Computed fields
    ReviewCount int     `bson:"review_count"`
    AvgRating   float64 `bson:"avg_rating"`
    TotalRating int     `bson:"total_rating"` // sum of all ratings
}

func (r *ProductRepo) AddReview(ctx context.Context, productID primitive.ObjectID, rating int) error {
    // Atomically update computed fields
    _, err := r.products.UpdateOne(ctx,
        bson.M{"_id": productID},
        bson.A{
            bson.M{"$set": bson.M{
                "total_rating": bson.M{"$add": bson.A{"$total_rating", rating}},
                "review_count": bson.M{"$add": bson.A{"$review_count", 1}},
                "avg_rating": bson.M{
                    "$divide": bson.A{
                        bson.M{"$add": bson.A{"$total_rating", rating}},
                        bson.M{"$add": bson.A{"$review_count", 1}},
                    },
                },
            }},
        },
    )
    return err
}
```

## BSON Struct Tags

### Основные теги

```go
type User struct {
    // "_id" — имя поля в MongoDB
    ID primitive.ObjectID `bson:"_id,omitempty"`

    // Простой маппинг имени
    FirstName string `bson:"first_name"`

    // omitempty — не записывать если zero value
    MiddleName string `bson:"middle_name,omitempty"`

    // "-" — полностью игнорировать поле
    InternalCache string `bson:"-"`

    // minsize — использовать минимальный BSON-тип для числа
    SmallNumber int64 `bson:"small_num,minsize"` // int32 if fits

    // truncate — обрезать время до миллисекунд (BSON DateTime precision)
    CreatedAt time.Time `bson:"created_at,truncate"`

    // inline — "разворачивает" вложенную структуру
    Address Address `bson:",inline"`
    // Результат: {street: "...", city: "..."} а не {address: {street: "...", city: "..."}}
}
```

### inline vs embedded

```go
// Embedded — вложенный документ
type UserEmbedded struct {
    ID      primitive.ObjectID `bson:"_id,omitempty"`
    Name    string             `bson:"name"`
    Address Address            `bson:"address"`
}
// BSON: {"_id": ..., "name": "Alice", "address": {"street": "Main St", "city": "Moscow"}}

// Inline — поля "разворачиваются" в родительский документ
type UserInline struct {
    ID      primitive.ObjectID `bson:"_id,omitempty"`
    Name    string             `bson:"name"`
    Address Address            `bson:",inline"`
}
// BSON: {"_id": ..., "name": "Alice", "street": "Main St", "city": "Moscow"}
```

### omitempty поведение для разных типов

```
| Тип | Zero value (не записывается с omitempty) |
|-----|------------------------------------------|
| string | "" |
| int, float | 0 |
| bool | false |
| time.Time | time.Time{} (zero time) |
| slice | nil (не пустой slice!) |
| pointer | nil |
| ObjectID | primitive.NilObjectID |
| map | nil |
```

**Важно**: `omitempty` для `[]string{}` (пустой slice, не nil) **запишет** пустой массив `[]`. Только `nil` slice пропускается.

```go
type Example struct {
    Tags []string `bson:"tags,omitempty"`
}

// nil slice — поле не записывается
e1 := Example{Tags: nil}
// BSON: {}

// empty slice — записывается пустой массив
e2 := Example{Tags: []string{}}
// BSON: {"tags": []}
```

## Custom Marshaling / Unmarshaling

### bson.Marshaler / bson.Unmarshaler

```go
type Money struct {
    Amount   int64  // в копейках
    Currency string
}

// MarshalBSON — кастомная сериализация в BSON
func (m Money) MarshalBSON() ([]byte, error) {
    return bson.Marshal(bson.M{
        "amount":   m.Amount,
        "currency": m.Currency,
    })
}

// UnmarshalBSON — кастомная десериализация из BSON
func (m *Money) UnmarshalBSON(data []byte) error {
    var raw struct {
        Amount   int64  `bson:"amount"`
        Currency string `bson:"currency"`
    }
    if err := bson.Unmarshal(data, &raw); err != nil {
        return err
    }
    m.Amount = raw.Amount
    m.Currency = raw.Currency
    return nil
}
```

### bson.ValueMarshaler / bson.ValueUnmarshaler

Для маршалинга в одно BSON-значение (не документ):

```go
type Status int

const (
    StatusActive   Status = 1
    StatusInactive Status = 2
    StatusBanned   Status = 3
)

var statusNames = map[Status]string{
    StatusActive:   "active",
    StatusInactive: "inactive",
    StatusBanned:   "banned",
}

var statusValues = map[string]Status{
    "active":   StatusActive,
    "inactive": StatusInactive,
    "banned":   StatusBanned,
}

// MarshalBSONValue — сохраняем как строку, а не число
func (s Status) MarshalBSONValue() (bsontype.Type, []byte, error) {
    name, ok := statusNames[s]
    if !ok {
        return 0, nil, fmt.Errorf("unknown status: %d", s)
    }
    return bson.MarshalValue(name)
}

// UnmarshalBSONValue — читаем строку и конвертируем обратно
func (s *Status) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
    if t != bsontype.String {
        return fmt.Errorf("expected string, got %v", t)
    }
    var name string
    if err := bson.UnmarshalValue(t, data, &name); err != nil {
        return err
    }
    val, ok := statusValues[name]
    if !ok {
        return fmt.Errorf("unknown status: %s", name)
    }
    *s = val
    return nil
}
```

## Лимит размера документа

MongoDB ограничивает размер документа **16 MB**. Это жёсткий лимит.

```
Что влияет на размер:
├── Длинные строковые поля
├── Встроенные массивы (растут бесконечно)
├── Binary данные (файлы, изображения)
└── Глубоко вложенные документы

Решения:
├── Referencing вместо embedding для растущих массивов
├── Bucket pattern для time-series
├── Subset pattern для массивов с ограничением
├── GridFS для файлов > 16 MB
└── Outlier pattern для редких больших документов
```

## Полиморфные документы

MongoDB позволяет хранить документы разной структуры в одной коллекции:

```go
// Разные типы уведомлений в одной коллекции
type Notification struct {
    ID        primitive.ObjectID `bson:"_id,omitempty"`
    Type      string             `bson:"type"`       // "email", "sms", "push"
    UserID    primitive.ObjectID `bson:"user_id"`
    CreatedAt time.Time          `bson:"created_at"`

    // Email-specific
    Subject string `bson:"subject,omitempty"`
    Body    string `bson:"body,omitempty"`

    // SMS-specific
    Phone   string `bson:"phone,omitempty"`
    Message string `bson:"message,omitempty"`

    // Push-specific
    DeviceToken string `bson:"device_token,omitempty"`
    Title       string `bson:"title,omitempty"`
}

// Альтернатива: использовать bson.Raw для гибкости
type Notification struct {
    ID        primitive.ObjectID `bson:"_id,omitempty"`
    Type      string             `bson:"type"`
    UserID    primitive.ObjectID `bson:"user_id"`
    Payload   bson.Raw           `bson:"payload"` // type-specific data
    CreatedAt time.Time          `bson:"created_at"`
}
```

## Типичные ошибки моделирования

```
1. Unbounded arrays (массивы без ограничений)
   ПЛОХО: хранить все комментарии к посту как embedded array
   Массив растёт бесконечно → документ достигает 16 MB
   РЕШЕНИЕ: referencing или subset pattern

2. Чрезмерная нормализация (как в SQL)
   ПЛОХО: отдельные коллекции для user, address, phone, email
   Каждый запрос требует несколько $lookup (JOIN)
   РЕШЕНИЕ: embedding для данных, которые читаются вместе

3. Чрезмерная денормализация
   ПЛОХО: копировать имя пользователя в каждый комментарий
   При изменении имени нужно обновить тысячи комментариев
   РЕШЕНИЕ: referencing для часто меняющихся данных

4. Моделирование по структуре приложения, а не по запросам
   ПЛОХО: "у меня есть класс User и класс Order → две коллекции"
   РЕШЕНИЕ: моделировать по паттернам доступа (query patterns)
```

---

## Вопросы на собеседовании

1. **Когда использовать embedding, а когда referencing?**
   Embedding — для связей 1:1 и 1:few, данных которые всегда читаются вместе, и массивов с ограниченным ростом. Referencing — для 1:many (тысячи), many:many, независимо запрашиваемых данных, и часто обновляемых вложенных данных. Ключевой критерий — паттерн доступа, а не структура данных.

2. **Какой максимальный размер документа в MongoDB и как обходить это ограничение?**
   16 MB. Обходить через: referencing (массивы в отдельной коллекции), subset pattern (только N последних элементов), bucket pattern (группировка time-series), GridFS (для файлов > 16 MB).

3. **Что такое Subset Pattern и когда его применять?**
   Хранить в документе только N последних/важных элементов массива, полную коллекцию — отдельно. Применять когда 90% запросов нуждаются только в последних элементах (последние отзывы, последние сообщения), а полный список нужен редко.

4. **Чем `bson:",inline"` отличается от обычного embedding?**
   `inline` разворачивает поля вложенной структуры в родительский документ. Обычное embedding создаёт вложенный документ. `inline` полезен для композиции (timestamp mixin, audit fields), но может вызвать конфликты имён.

5. **Как `omitempty` работает с разными типами?**
   Пропускает zero values: "" для string, 0 для чисел, nil для pointer/slice/map. Важный нюанс: пустой slice (`[]string{}`) НЕ пропускается — только nil slice. Для `bool` zero value — это `false`, что может быть неожиданным.

6. **Как моделировать many-to-many связь в MongoDB?**
   Два подхода: (1) массив ссылок в одном из документов (`tags: [id1, id2, id3]`) — если количество связей ограничено; (2) отдельная коллекция связей (`{user_id, group_id}`) — если связей очень много или нужны метаданные связи (дата добавления, роль).
