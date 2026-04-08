# Продвинутый SQL

## Обзор

CTE, оконные функции, JSONB, UPSERT, bulk insert, lateral joins — продвинутые возможности PostgreSQL, которые часто встречаются в реальных проектах и на собеседованиях. Понимание этих инструментов отличает middle от junior.

## CTE (Common Table Expressions)

CTE — именованные временные результаты запросов. Улучшают читаемость и позволяют переиспользовать промежуточные результаты.

### Базовый синтаксис

```sql
-- Без CTE: вложенные подзапросы, тяжело читать
SELECT * FROM (
    SELECT user_id, SUM(total) AS total_spent
    FROM orders
    WHERE created_at > '2024-01-01'
    GROUP BY user_id
) AS spending
WHERE total_spent > 1000;

-- С CTE: читаемо и декларативно
WITH spending AS (
    SELECT user_id, SUM(total) AS total_spent
    FROM orders
    WHERE created_at > '2024-01-01'
    GROUP BY user_id
)
SELECT u.name, s.total_spent
FROM spending s
JOIN users u ON u.id = s.user_id
WHERE s.total_spent > 1000;
```

### Несколько CTE

```sql
WITH
active_users AS (
    SELECT id, name FROM users WHERE active = true
),
user_orders AS (
    SELECT user_id, COUNT(*) AS order_count, SUM(total) AS total_spent
    FROM orders
    GROUP BY user_id
)
SELECT au.name, uo.order_count, uo.total_spent
FROM active_users au
JOIN user_orders uo ON uo.user_id = au.id
ORDER BY uo.total_spent DESC;
```

### Materialization hints (PostgreSQL 12+)

```sql
-- MATERIALIZED — CTE вычисляется один раз и результат кэшируется
-- Полезно если CTE используется несколько раз
WITH heavy_query AS MATERIALIZED (
    SELECT user_id, complex_calculation(data) AS result
    FROM big_table
    WHERE some_condition
)
SELECT * FROM heavy_query WHERE result > 100
UNION ALL
SELECT * FROM heavy_query WHERE result < 10;

-- NOT MATERIALIZED — оптимизатор может "встроить" CTE в основной запрос
-- По умолчанию для CTE, используемых один раз (PG12+)
WITH simple_filter AS NOT MATERIALIZED (
    SELECT * FROM users WHERE active = true
)
SELECT * FROM simple_filter WHERE name LIKE 'A%';
-- Оптимизатор может объединить: SELECT * FROM users WHERE active = true AND name LIKE 'A%'
```

## Recursive CTE

### Обход дерева (иерархия сотрудников)

```sql
CREATE TABLE employees (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    manager_id INT REFERENCES employees(id)
);

-- Найти всех подчинённых менеджера (рекурсивно)
WITH RECURSIVE subordinates AS (
    -- Base case: сам менеджер
    SELECT id, name, manager_id, 0 AS depth
    FROM employees
    WHERE id = 1

    UNION ALL

    -- Recursive case: подчинённые подчинённых
    SELECT e.id, e.name, e.manager_id, s.depth + 1
    FROM employees e
    JOIN subordinates s ON e.manager_id = s.id
)
SELECT * FROM subordinates ORDER BY depth, name;
```

### Обход категорий (вложенные категории)

```sql
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    parent_id INT REFERENCES categories(id)
);

-- Построить полный путь для каждой категории
WITH RECURSIVE cat_path AS (
    SELECT id, name, parent_id, name::TEXT AS full_path
    FROM categories
    WHERE parent_id IS NULL  -- корневые категории

    UNION ALL

    SELECT c.id, c.name, c.parent_id, cp.full_path || ' > ' || c.name
    FROM categories c
    JOIN cat_path cp ON c.parent_id = cp.id
)
SELECT * FROM cat_path ORDER BY full_path;
-- "Электроника"
-- "Электроника > Смартфоны"
-- "Электроника > Смартфоны > Apple"
```

### Генерация последовательностей

```sql
-- Генерация дат (календарь)
WITH RECURSIVE dates AS (
    SELECT '2024-01-01'::DATE AS d
    UNION ALL
    SELECT d + 1 FROM dates WHERE d < '2024-01-31'
)
SELECT d, EXTRACT(DOW FROM d) AS day_of_week
FROM dates;

-- Числа Фибоначчи
WITH RECURSIVE fib AS (
    SELECT 1 AS n, 1::BIGINT AS val, 0::BIGINT AS prev
    UNION ALL
    SELECT n + 1, val + prev, val FROM fib WHERE n < 20
)
SELECT n, val FROM fib;
```

## Оконные функции

Оконные функции выполняют вычисления по набору строк, связанных с текущей строкой, **без группировки** (в отличие от GROUP BY).

### ROW_NUMBER, RANK, DENSE_RANK

```sql
-- Данные: оценки студентов
-- | student | subject | score |
-- |---------|---------|-------|
-- | Alice   | Math    | 95    |
-- | Bob     | Math    | 90    |
-- | Carol   | Math    | 90    |
-- | Dave    | Math    | 85    |

SELECT
    student,
    subject,
    score,
    ROW_NUMBER() OVER (ORDER BY score DESC) AS row_num,   -- 1, 2, 3, 4 (всегда уникальный)
    RANK()       OVER (ORDER BY score DESC) AS rank,       -- 1, 2, 2, 4 (пропускает после дубля)
    DENSE_RANK() OVER (ORDER BY score DESC) AS dense_rank  -- 1, 2, 2, 3 (не пропускает)
FROM scores
WHERE subject = 'Math';
```

### LAG и LEAD

```sql
-- Доступ к предыдущей/следующей строке
SELECT
    date,
    revenue,
    LAG(revenue, 1)  OVER (ORDER BY date) AS prev_day_revenue,
    LEAD(revenue, 1) OVER (ORDER BY date) AS next_day_revenue,
    revenue - LAG(revenue, 1) OVER (ORDER BY date) AS daily_change
FROM daily_revenue;

-- LAG(value, offset, default) — offset и default опциональны
-- LAG(revenue, 7) — выручка неделю назад
```

### Агрегация без GROUP BY

```sql
-- Кумулятивная сумма
SELECT
    date,
    revenue,
    SUM(revenue)  OVER (ORDER BY date) AS cumulative_revenue,
    AVG(revenue)  OVER (ORDER BY date ROWS BETWEEN 6 PRECEDING AND CURRENT ROW) AS moving_avg_7d,
    COUNT(*)      OVER () AS total_rows  -- по всему набору
FROM daily_revenue;
```

### PARTITION BY + ORDER BY

```sql
-- Ранжирование внутри групп
SELECT
    department,
    employee,
    salary,
    RANK()         OVER (PARTITION BY department ORDER BY salary DESC) AS dept_rank,
    SUM(salary)    OVER (PARTITION BY department) AS dept_total,
    salary::FLOAT / SUM(salary) OVER (PARTITION BY department) * 100 AS salary_pct
FROM employees;
```

### Рамки окна (Window Frames)

```sql
-- ROWS BETWEEN — считает физические строки
SELECT
    date,
    revenue,
    -- Скользящее среднее за 7 дней
    AVG(revenue) OVER (
        ORDER BY date
        ROWS BETWEEN 6 PRECEDING AND CURRENT ROW
    ) AS moving_avg_7d,

    -- Сумма: от начала до текущей строки
    SUM(revenue) OVER (
        ORDER BY date
        ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
    ) AS running_total
FROM daily_revenue;

-- RANGE BETWEEN — считает логические значения (по значению ORDER BY)
-- Полезно для дат с пропусками
SELECT
    date,
    revenue,
    SUM(revenue) OVER (
        ORDER BY date
        RANGE BETWEEN INTERVAL '7 days' PRECEDING AND CURRENT ROW
    ) AS week_total
FROM daily_revenue;
```

## JSONB

### Операторы

```sql
CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    data JSONB NOT NULL
);

-- Вставка
INSERT INTO events (data) VALUES ('{"type": "click", "page": "/home", "meta": {"browser": "Chrome", "os": "Linux"}}');

-- Операторы доступа
SELECT
    data -> 'type'           AS type_json,      -- "click" (JSON)
    data ->> 'type'          AS type_text,      -- click (text)
    data -> 'meta' -> 'browser' AS browser_json, -- "Chrome" (JSON)
    data #> '{meta,browser}' AS browser_path,    -- "Chrome" (JSON, по пути)
    data #>> '{meta,os}'     AS os_text          -- Linux (text, по пути)
FROM events;

-- Операторы фильтрации
SELECT * FROM events WHERE data @> '{"type": "click"}';           -- containment
SELECT * FROM events WHERE data ? 'type';                          -- ключ существует
SELECT * FROM events WHERE data ?| array['type', 'action'];        -- любой ключ существует
SELECT * FROM events WHERE data ?& array['type', 'page'];          -- все ключи существуют
SELECT * FROM events WHERE data ->> 'type' = 'click';              -- сравнение по значению
SELECT * FROM events WHERE (data ->> 'score')::INT > 50;           -- приведение типа
```

### Функции

```sql
-- Построение JSON
SELECT jsonb_build_object(
    'name', u.name,
    'email', u.email,
    'orders_count', COUNT(o.id)
)
FROM users u
LEFT JOIN orders o ON o.user_id = u.id
GROUP BY u.id;

-- Агрегация в JSON-массив
SELECT
    u.name,
    jsonb_agg(jsonb_build_object('id', o.id, 'total', o.total)) AS orders
FROM users u
JOIN orders o ON o.user_id = u.id
GROUP BY u.id;

-- Разбор JSON-массива
SELECT * FROM jsonb_each('{"a": 1, "b": 2}'::JSONB);        -- key/value пары
SELECT * FROM jsonb_each_text('{"a": 1, "b": 2}'::JSONB);    -- key/value (text)

-- JSON → таблица
SELECT * FROM jsonb_to_recordset(
    '[{"name": "Alice", "age": 30}, {"name": "Bob", "age": 25}]'::JSONB
) AS t(name TEXT, age INT);
```

### Индексирование JSONB

```sql
-- jsonb_ops (default) — поддерживает @>, ?, ?|, ?&
CREATE INDEX idx_events_data ON events USING GIN (data);

-- jsonb_path_ops — только @> (containment), но меньше и быстрее
CREATE INDEX idx_events_data_path ON events USING GIN (data jsonb_path_ops);

-- Индекс на конкретное поле
CREATE INDEX idx_events_type ON events ((data ->> 'type'));
```

### Когда JSONB vs отдельные столбцы

| Используй JSONB | Используй столбцы |
|-----------------|-------------------|
| Схема динамическая, меняется | Схема стабильная |
| Данные от внешних систем (API) | Часто фильтруешь/сортируешь |
| Мета-информация, конфигурация | Нужны UNIQUE/FK constraints |
| Вложенные структуры | Критичен перфоманс запросов |
| Редко запрашиваемые атрибуты | Часто JOIN-ишь по полям |

## UPSERT

`INSERT ... ON CONFLICT` — атомарная вставка или обновление.

```sql
-- ON CONFLICT DO UPDATE (upsert)
INSERT INTO users (email, name, updated_at)
VALUES ('alice@example.com', 'Alice', NOW())
ON CONFLICT (email)
DO UPDATE SET
    name = EXCLUDED.name,
    updated_at = EXCLUDED.updated_at;

-- EXCLUDED — специальная таблица с данными, которые пытались вставить

-- ON CONFLICT DO NOTHING — пропустить дубликаты
INSERT INTO user_views (user_id, page, viewed_at)
VALUES (1, '/home', NOW())
ON CONFLICT (user_id, page) DO NOTHING;

-- Условный upsert
INSERT INTO counters (key, value)
VALUES ('page_views', 1)
ON CONFLICT (key)
DO UPDATE SET value = counters.value + EXCLUDED.value
WHERE counters.updated_at < NOW() - INTERVAL '1 minute'; -- обновлять не чаще раза в минуту

-- Upsert с RETURNING
INSERT INTO users (email, name)
VALUES ('alice@example.com', 'Alice')
ON CONFLICT (email)
DO UPDATE SET name = EXCLUDED.name
RETURNING id, (xmax = 0) AS inserted;  -- true если INSERT, false если UPDATE
```

### UPSERT в Go

```go
func upsertUser(ctx context.Context, pool *pgxpool.Pool, email, name string) (int64, error) {
    var id int64
    err := pool.QueryRow(ctx, `
        INSERT INTO users (email, name)
        VALUES ($1, $2)
        ON CONFLICT (email)
        DO UPDATE SET name = EXCLUDED.name
        RETURNING id
    `, email, name).Scan(&id)
    return id, err
}
```

## Bulk Insert

### Batch INSERT

```go
// Обычный batch INSERT — несколько VALUES в одном запросе
func batchInsert(ctx context.Context, pool *pgxpool.Pool, users []User) error {
    query := "INSERT INTO users (name, email) VALUES "
    var args []any
    for i, u := range users {
        if i > 0 {
            query += ", "
        }
        query += fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2)
        args = append(args, u.Name, u.Email)
    }
    _, err := pool.Exec(ctx, query, args...)
    return err
}
```

### COPY — максимальная скорость

```go
// pgx.CopyFrom — использует COPY протокол PostgreSQL
// В 5-10 раз быстрее batch INSERT для больших объёмов
func bulkInsertCopy(ctx context.Context, pool *pgxpool.Pool, users []User) error {
    rows := make([][]any, len(users))
    for i, u := range users {
        rows[i] = []any{u.Name, u.Email}
    }

    _, err := pool.CopyFrom(
        ctx,
        pgx.Identifier{"users"},           // таблица
        []string{"name", "email"},          // столбцы
        pgx.CopyFromRows(rows),            // данные
    )
    return err
}

// CopyFrom с интерфейсом CopyFromSource (для streaming)
type userSource struct {
    users []User
    idx   int
}

func (s *userSource) Next() bool {
    s.idx++
    return s.idx < len(s.users)
}

func (s *userSource) Values() ([]any, error) {
    u := s.users[s.idx]
    return []any{u.Name, u.Email}, nil
}

func (s *userSource) Err() error { return nil }
```

**Сравнение скорости** (вставка 100 000 строк):

| Метод | Время |
|-------|-------|
| INSERT по одной строке | ~30 сек |
| Batch INSERT (1000 VALUES) | ~3 сек |
| COPY | ~0.5 сек |

## Lateral Joins

`LATERAL` позволяет подзапросу ссылаться на столбцы из предшествующих таблиц — как коррелированный подзапрос, но в FROM.

```sql
-- Задача: для каждого пользователя найти 3 последних заказа
-- Без LATERAL: сложно и неэффективно

-- С LATERAL:
SELECT u.name, o.id, o.total, o.created_at
FROM users u
LEFT JOIN LATERAL (
    SELECT id, total, created_at
    FROM orders
    WHERE orders.user_id = u.id  -- доступ к u.id из внешней таблицы!
    ORDER BY created_at DESC
    LIMIT 3
) o ON true;

-- Для каждого пользователя — ближайший магазин
SELECT u.name, s.store_name, s.distance
FROM users u
LEFT JOIN LATERAL (
    SELECT
        name AS store_name,
        ST_Distance(u.location, stores.location) AS distance
    FROM stores
    ORDER BY u.location <-> stores.location
    LIMIT 1
) s ON true;
```

**LATERAL vs обычный подзапрос в FROM**: обычный подзапрос в FROM не может ссылаться на другие таблицы в том же FROM. LATERAL снимает это ограничение.

### LATERAL в Go

```go
func getRecentOrdersPerUser(ctx context.Context, pool *pgxpool.Pool) ([]UserOrders, error) {
    rows, err := pool.Query(ctx, `
        SELECT u.id, u.name, o.id, o.total, o.created_at
        FROM users u
        LEFT JOIN LATERAL (
            SELECT id, total, created_at
            FROM orders
            WHERE orders.user_id = u.id
            ORDER BY created_at DESC
            LIMIT 3
        ) o ON true
        ORDER BY u.id, o.created_at DESC
    `)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    // Сканирование и группировка результатов...
    var result []UserOrders
    // ...
    return result, nil
}
```

## Вопросы для собеседования

1. **Напишите запрос для top-3 per group (3 самых дорогих заказа для каждого пользователя).**

```sql
-- Вариант 1: ROW_NUMBER
WITH ranked AS (
    SELECT
        user_id,
        id,
        total,
        ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY total DESC) AS rn
    FROM orders
)
SELECT * FROM ranked WHERE rn <= 3;

-- Вариант 2: LATERAL (часто эффективнее с индексом на (user_id, total DESC))
SELECT u.id, o.id, o.total
FROM users u
LEFT JOIN LATERAL (
    SELECT id, total
    FROM orders
    WHERE orders.user_id = u.id
    ORDER BY total DESC
    LIMIT 3
) o ON true;
```

2. **Как работает UPSERT в PostgreSQL?**
   `INSERT ... ON CONFLICT (column) DO UPDATE SET ...` — атомарная операция: если строка с таким значением уже существует (конфликт по unique constraint), выполняется UPDATE вместо INSERT. `EXCLUDED` — виртуальная таблица с данными, которые пытались вставить. `ON CONFLICT DO NOTHING` — просто пропускает дубликаты.

3. **Чем CTE отличается от подзапроса?**
   CTE (`WITH ... AS`) именует результат запроса и может быть переиспользован. До PG12 CTE всегда материализовался (вычислялся отдельно). С PG12+ CTE, используемый один раз, может быть "встроен" в основной запрос оптимизатором (NOT MATERIALIZED). Подзапрос всегда оптимизируется совместно с основным запросом.

4. **Когда использовать JSONB вместо отдельных столбцов?**
   JSONB хорош для: динамической схемы, данных от внешних API, мета-информации, вложенных структур. Отдельные столбцы лучше когда: схема стабильна, нужны constraint-ы (UNIQUE, FK), часто фильтруешь или сортируешь по полям, критична производительность.

5. **Что такое LATERAL JOIN и когда он нужен?**
   LATERAL позволяет подзапросу в FROM ссылаться на столбцы из предшествующих таблиц. Нужен для задач "top-N per group", "ближайший объект для каждой строки" — везде, где нужен коррелированный подзапрос с LIMIT, ORDER BY или агрегацией.

6. **Почему COPY быстрее batch INSERT?**
   COPY использует специальный бинарный протокол PostgreSQL, минуя парсинг SQL. Нет overhead-а на разбор отдельных INSERT, проверку синтаксиса, планирование запроса. Данные передаются потоком. Для 100K+ строк COPY в 5-10 раз быстрее batch INSERT.
