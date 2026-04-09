# MongoDB: Агрегации

## Aggregation Pipeline

Aggregation pipeline — последовательность стадий (stages), каждая из которых трансформирует поток документов. Аналог SQL GROUP BY + подзапросы + оконные функции.

```
Коллекция → [$match] → [$group] → [$sort] → [$limit] → Результат

Каждая стадия:
1. Получает поток документов
2. Трансформирует их
3. Передаёт результат следующей стадии
```

### Базовый пример в Go

```go
pipeline := bson.A{
    bson.M{"$match": bson.M{"status": "active"}},
    bson.M{"$group": bson.M{
        "_id":        "$city",
        "total":      bson.M{"$sum": "$amount"},
        "avg_amount": bson.M{"$avg": "$amount"},
        "count":      bson.M{"$sum": 1},
    }},
    bson.M{"$sort": bson.M{"total": -1}},
    bson.M{"$limit": 10},
}

cursor, err := coll.Aggregate(ctx, pipeline)
if err != nil {
    return nil, fmt.Errorf("aggregate: %w", err)
}
defer cursor.Close(ctx)

var results []CityStats
if err := cursor.All(ctx, &results); err != nil {
    return nil, err
}
```

## Основные стадии

### $match

Фильтрация документов. Аналог WHERE в SQL. Ставить как можно раньше в pipeline для оптимизации.

```go
// Простой фильтр
bson.M{"$match": bson.M{"status": "active"}}

// С операторами
bson.M{"$match": bson.M{
    "created_at": bson.M{"$gte": startDate, "$lt": endDate},
    "amount":     bson.M{"$gt": 100},
}}

// $match в начале pipeline может использовать индексы!
// Дальше по pipeline — только in-memory обработка
```

### $group

Группировка документов. Аналог GROUP BY в SQL.

```go
// Группировка по городу с агрегатными функциями
bson.M{"$group": bson.M{
    "_id":   "$city",                              // поле группировки
    "total": bson.M{"$sum": "$amount"},            // SUM(amount)
    "avg":   bson.M{"$avg": "$amount"},            // AVG(amount)
    "min":   bson.M{"$min": "$amount"},            // MIN(amount)
    "max":   bson.M{"$max": "$amount"},            // MAX(amount)
    "count": bson.M{"$sum": 1},                    // COUNT(*)
    "items": bson.M{"$push": "$name"},             // собрать в массив
    "first": bson.M{"$first": "$name"},            // первый элемент
    "last":  bson.M{"$last": "$name"},             // последний элемент
    "unique": bson.M{"$addToSet": "$category"},    // уникальные значения
}}

// Группировка по нескольким полям
bson.M{"$group": bson.M{
    "_id": bson.M{
        "city":   "$city",
        "status": "$status",
    },
    "count": bson.M{"$sum": 1},
}}

// Без группировки (_id: null) — агрегация по всей коллекции
bson.M{"$group": bson.M{
    "_id":   nil,
    "total": bson.M{"$sum": "$amount"},
    "count": bson.M{"$sum": 1},
}}
```

### $project

Выбор/переименование/вычисление полей. Аналог SELECT в SQL.

```go
bson.M{"$project": bson.M{
    "name":  1,                                    // включить
    "email": 1,                                    // включить
    "_id":   0,                                    // исключить

    // Вычисляемое поле
    "full_name": bson.M{
        "$concat": bson.A{"$first_name", " ", "$last_name"},
    },

    // Условное поле
    "tier": bson.M{
        "$switch": bson.M{
            "branches": bson.A{
                bson.M{"case": bson.M{"$gte": bson.A{"$total_spent", 10000}}, "then": "gold"},
                bson.M{"case": bson.M{"$gte": bson.A{"$total_spent", 5000}}, "then": "silver"},
            },
            "default": "bronze",
        },
    },

    // Математика
    "price_with_tax": bson.M{"$multiply": bson.A{"$price", 1.2}},
    "discount_price": bson.M{"$subtract": bson.A{"$price", "$discount"}},
}}
```

### $sort

```go
// Сортировка: 1 = ASC, -1 = DESC
bson.M{"$sort": bson.M{"total": -1, "name": 1}}

// $sort в начале pipeline (после $match) может использовать индексы
// $sort позже — in-memory, лимит 100 MB
// Если > 100 MB, нужен allowDiskUse
```

### $limit и $skip

```go
bson.M{"$skip": 20}   // пропустить 20 документов
bson.M{"$limit": 10}  // взять 10 документов

// Порядок важен: $sort → $skip → $limit
```

### $unwind

Разворачивает массив — создаёт отдельный документ для каждого элемента:

```go
// Документ: {name: "Alice", tags: ["go", "mongo", "docker"]}
// После $unwind:
// {name: "Alice", tags: "go"}
// {name: "Alice", tags: "mongo"}
// {name: "Alice", tags: "docker"}

bson.M{"$unwind": "$tags"}

// С опциями — сохранить документы без массива или с пустым массивом
bson.M{"$unwind": bson.M{
    "path":                       "$tags",
    "preserveNullAndEmptyArrays": true,  // не отбрасывать документы без tags
    "includeArrayIndex":          "idx", // добавить поле с индексом элемента
}}
```

### $lookup (JOIN)

Объединение данных из другой коллекции. Аналог LEFT JOIN в SQL.

```go
// Простой lookup: найти все заказы для каждого пользователя
bson.M{"$lookup": bson.M{
    "from":         "orders",    // коллекция для JOIN
    "localField":   "_id",       // поле текущей коллекции
    "foreignField": "user_id",   // поле в orders
    "as":           "orders",    // имя результирующего массива
}}
// Результат: каждый user получает поле orders: [{...}, {...}]

// Pipeline lookup — более гибкий вариант
bson.M{"$lookup": bson.M{
    "from": "orders",
    "let":  bson.M{"userId": "$_id"}, // переменные из текущего документа
    "pipeline": bson.A{
        bson.M{"$match": bson.M{
            "$expr": bson.M{
                "$and": bson.A{
                    bson.M{"$eq": bson.A{"$user_id", "$$userId"}},
                    bson.M{"$eq": bson.A{"$status", "completed"}},
                },
            },
        }},
        bson.M{"$sort": bson.M{"created_at": -1}},
        bson.M{"$limit": 5},
    },
    "as": "recent_orders",
}}
```

### Полный пример: $lookup + $unwind

```go
// Отчёт: пользователи с суммой их заказов
func (r *ReportRepo) UserOrderStats(ctx context.Context) ([]UserStats, error) {
    pipeline := bson.A{
        // JOIN с заказами
        bson.M{"$lookup": bson.M{
            "from":         "orders",
            "localField":   "_id",
            "foreignField": "user_id",
            "as":           "orders",
        }},
        // Развернуть массив заказов
        bson.M{"$unwind": bson.M{
            "path":                       "$orders",
            "preserveNullAndEmptyArrays": true,
        }},
        // Группировка по пользователю
        bson.M{"$group": bson.M{
            "_id":         "$_id",
            "name":        bson.M{"$first": "$name"},
            "email":       bson.M{"$first": "$email"},
            "total_spent": bson.M{"$sum": "$orders.amount"},
            "order_count": bson.M{"$sum": bson.M{
                "$cond": bson.A{
                    bson.M{"$ifNull": bson.A{"$orders", false}},
                    1, 0,
                },
            }},
        }},
        // Сортировка по сумме
        bson.M{"$sort": bson.M{"total_spent": -1}},
    }

    cursor, err := r.users.Aggregate(ctx, pipeline)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var stats []UserStats
    return stats, cursor.All(ctx, &stats)
}

type UserStats struct {
    ID         primitive.ObjectID `bson:"_id"`
    Name       string             `bson:"name"`
    Email      string             `bson:"email"`
    TotalSpent float64            `bson:"total_spent"`
    OrderCount int                `bson:"order_count"`
}
```

### $addFields

Добавление новых полей без изменения существующих:

```go
bson.M{"$addFields": bson.M{
    "total_with_tax": bson.M{"$multiply": bson.A{"$total", 1.2}},
    "year":           bson.M{"$year": "$created_at"},
    "is_premium":     bson.M{"$gte": bson.A{"$total_spent", 10000}},
}}
```

### $facet

Несколько pipeline за один запрос — полезно для страниц с фильтрами и счётчиками:

```go
// Одновременно: результаты + общее количество + статистика по категориям
pipeline := bson.A{
    bson.M{"$match": bson.M{"status": "active"}},
    bson.M{"$facet": bson.M{
        // Пагинированные результаты
        "items": bson.A{
            bson.M{"$sort": bson.M{"created_at": -1}},
            bson.M{"$skip": offset},
            bson.M{"$limit": limit},
        },
        // Общее количество
        "total_count": bson.A{
            bson.M{"$count": "count"},
        },
        // Агрегация по категориям
        "by_category": bson.A{
            bson.M{"$group": bson.M{
                "_id":   "$category",
                "count": bson.M{"$sum": 1},
            }},
            bson.M{"$sort": bson.M{"count": -1}},
        },
    }},
}

cursor, err := coll.Aggregate(ctx, pipeline)
```

Результат `$facet`:

```json
{
    "items": [/* массив документов */],
    "total_count": [{"count": 1234}],
    "by_category": [
        {"_id": "electronics", "count": 500},
        {"_id": "books", "count": 300}
    ]
}
```

### $bucket и $bucketAuto

Распределение по диапазонам:

```go
// $bucket — ручные границы
bson.M{"$bucket": bson.M{
    "groupBy":    "$age",
    "boundaries": bson.A{0, 18, 30, 50, 100},
    "default":    "other",
    "output": bson.M{
        "count": bson.M{"$sum": 1},
        "names": bson.M{"$push": "$name"},
    },
}}
// Результат: {_id: 0, count: 5}, {_id: 18, count: 120}, ...

// $bucketAuto — автоматические границы
bson.M{"$bucketAuto": bson.M{
    "groupBy": "$price",
    "buckets": 5, // разбить на 5 равных групп
}}
```

### $count

```go
bson.M{"$count": "total"} // { total: 42 }
```

### $out и $merge

Запись результатов в коллекцию:

```go
// $out — полностью заменяет коллекцию результатом
bson.M{"$out": "monthly_report"}

// $merge — вставить/обновить в существующую коллекцию
bson.M{"$merge": bson.M{
    "into":        "stats",
    "on":          "_id",           // match field
    "whenMatched": "merge",         // merge, replace, keepExisting, fail
    "whenNotMatched": "insert",     // insert, discard, fail
}}
```

## Cursor и обработка результатов

### Итерация по курсору

```go
cursor, err := coll.Aggregate(ctx, pipeline)
if err != nil {
    return err
}
defer cursor.Close(ctx) // ОБЯЗАТЕЛЬНО закрыть

// Вариант 1: cursor.All() — загрузить всё в память
var results []bson.M
if err := cursor.All(ctx, &results); err != nil {
    return err
}

// Вариант 2: итерация — для больших результатов
for cursor.Next(ctx) {
    var doc bson.M
    if err := cursor.Decode(&doc); err != nil {
        return err
    }
    // process doc...
}
if err := cursor.Err(); err != nil {
    return err
}
```

### AllowDiskUse

Если pipeline превышает 100 MB RAM:

```go
opts := options.Aggregate().SetAllowDiskUse(true)
cursor, err := coll.Aggregate(ctx, pipeline, opts)
```

### BatchSize

Контроль размера batch при получении данных с сервера:

```go
opts := options.Aggregate().SetBatchSize(100)
cursor, err := coll.Aggregate(ctx, pipeline, opts)
```

## Оптимизация pipeline

### Порядок стадий

```
1. $match первым — фильтрует данные, может использовать индексы
2. $project раньше $group — уменьшить размер документов ДО группировки
3. $sort после $match — меньше данных для сортировки
4. $limit как можно раньше
```

### MongoDB автоматические оптимизации

```
MongoDB оптимизирует pipeline автоматически:
├── $match + $match → объединяет в один $match
├── $sort + $limit → использует top-N sort (эффективнее)
├── $match перед $lookup → перемещает match до lookup если возможно
└── $project + $match → переставляет match раньше если возможно
```

## Практический пример: аналитический отчёт

```go
// Ежемесячный отчёт продаж по категориям
func (r *ReportRepo) MonthlySales(ctx context.Context, year int) ([]MonthlySales, error) {
    startOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
    endOfYear := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)

    pipeline := bson.A{
        // Filter by year
        bson.M{"$match": bson.M{
            "created_at": bson.M{
                "$gte": startOfYear,
                "$lt":  endOfYear,
            },
            "status": "completed",
        }},
        // Group by month and category
        bson.M{"$group": bson.M{
            "_id": bson.M{
                "month":    bson.M{"$month": "$created_at"},
                "category": "$category",
            },
            "revenue": bson.M{"$sum": "$total"},
            "orders":  bson.M{"$sum": 1},
            "avg_order": bson.M{"$avg": "$total"},
        }},
        // Sort by month
        bson.M{"$sort": bson.D{
            {"_id.month", 1},
            {"revenue", -1},
        }},
        // Reshape output
        bson.M{"$project": bson.M{
            "_id":       0,
            "month":     "$_id.month",
            "category":  "$_id.category",
            "revenue":   1,
            "orders":    1,
            "avg_order": bson.M{"$round": bson.A{"$avg_order", 2}},
        }},
    }

    opts := options.Aggregate().SetAllowDiskUse(true)
    cursor, err := r.orders.Aggregate(ctx, pipeline, opts)
    if err != nil {
        return nil, fmt.Errorf("aggregate monthly sales: %w", err)
    }
    defer cursor.Close(ctx)

    var results []MonthlySales
    return results, cursor.All(ctx, &results)
}

type MonthlySales struct {
    Month    int     `bson:"month"`
    Category string  `bson:"category"`
    Revenue  float64 `bson:"revenue"`
    Orders   int     `bson:"orders"`
    AvgOrder float64 `bson:"avg_order"`
}
```

## Типичные ошибки

```
1. $match не в начале pipeline — не используются индексы
   РЕШЕНИЕ: всегда ставить $match первым

2. $lookup без лимита — загружает ВСЕ связанные документы
   Если у пользователя 100000 заказов — все загрузятся в один документ
   РЕШЕНИЕ: использовать pipeline-вариант $lookup с $limit

3. Забыли cursor.Close() — утечка ресурсов на сервере
   РЕШЕНИЕ: defer cursor.Close(ctx) сразу после Aggregate

4. cursor.All() на огромном результате — OOM
   РЕШЕНИЕ: итерация через cursor.Next() или $limit

5. In-memory sort > 100 MB — ошибка
   РЕШЕНИЕ: SetAllowDiskUse(true) или добавить индекс для $sort
```

---

## Вопросы на собеседовании

1. **Что такое aggregation pipeline в MongoDB?**
   Последовательность стадий, каждая из которых трансформирует поток документов. Документы проходят через стадии одна за другой. Аналог SQL GROUP BY + подзапросы, но более гибкий — можно комбинировать фильтрацию, группировку, JOIN, проекцию и вычисления.

2. **Как работает $lookup и чем он отличается от SQL JOIN?**
   `$lookup` выполняет LEFT OUTER JOIN с другой коллекцией. Результат всегда массив (даже если один или ноль совпадений). В отличие от SQL, в MongoDB нет INNER JOIN / RIGHT JOIN как отдельных операций — это реализуется через $lookup + $unwind + $match.

3. **Зачем нужен $unwind?**
   Разворачивает массив в отдельные документы — по одному на каждый элемент. Часто используется после `$lookup` для дальнейшей группировки или фильтрации. Без `$unwind` нельзя агрегировать по элементам массива.

4. **Что делает $facet и когда его использовать?**
   `$facet` выполняет несколько sub-pipeline параллельно на одних и тех же входных данных. Полезен для страниц с фильтрами: одновременно получить результаты, общее количество и агрегации по категориям за один запрос к серверу.

5. **Как оптимизировать aggregation pipeline?**
   (1) `$match` первым — для использования индексов; (2) `$project` до `$group` — уменьшить размер документов; (3) `$limit` как можно раньше; (4) избегать `$lookup` без лимитов; (5) `allowDiskUse` для тяжёлых pipeline; (6) использовать `$merge`/`$out` для предвычисленных отчётов.

6. **Чем $out отличается от $merge?**
   `$out` полностью заменяет целевую коллекцию результатом pipeline (drop + rename). `$merge` гибко вставляет/обновляет документы в существующую коллекцию с контролем поведения при совпадении/несовпадении. `$merge` может писать в ту же коллекцию, `$out` — нет.
