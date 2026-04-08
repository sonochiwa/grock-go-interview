# Индексы и EXPLAIN

## Обзор

Индексы — основной инструмент оптимизации запросов в PostgreSQL. Понимание типов индексов, умение читать планы выполнения и мониторить использование индексов — ключевые навыки для backend-разработчика.

## EXPLAIN ANALYZE

### Базовый синтаксис

```sql
EXPLAIN ANALYZE SELECT * FROM users WHERE email = 'alice@example.com';
```

`EXPLAIN` показывает план запроса. `ANALYZE` реально выполняет запрос и добавляет фактические метрики.

### Формат вывода

```sql
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT * FROM orders WHERE user_id = 42;

-- Результат:
-- Index Scan using idx_orders_user_id on orders  (cost=0.43..8.45 rows=5 width=48)
--                                                 (actual time=0.023..0.031 rows=3 loops=1)
--   Index Cond: (user_id = 42)
--   Buffers: shared hit=4
-- Planning Time: 0.085 ms
-- Execution Time: 0.052 ms
```

### Ключевые метрики

| Метрика | Значение |
|---------|----------|
| `cost=0.43..8.45` | Стоимость: startup..total (условные единицы) |
| `rows=5` | Ожидаемое количество строк (оценка планировщика) |
| `actual time=0.023..0.031` | Реальное время: первая строка..последняя (мс) |
| `rows=3` (actual) | Реальное количество строк |
| `loops=1` | Сколько раз узел выполнялся |
| `Buffers: shared hit=4` | Страниц прочитано из кэша (hit) или с диска (read) |

**Важно**: если `rows` (estimated) сильно отличается от `rows` (actual) — статистика устарела, нужен `ANALYZE table_name`.

## Типы сканирования

### Seq Scan — последовательное сканирование

```sql
-- Читает ВСЮ таблицу от начала до конца
EXPLAIN SELECT * FROM users WHERE name LIKE '%alice%';

-- Seq Scan on users  (cost=0.00..125.00 rows=50 width=64)
--   Filter: (name ~~ '%alice%')
--   Rows Removed by Filter: 9950
```

Seq Scan — не всегда плохо. Для маленьких таблиц или при выборке большой доли строк (>5-10%) это может быть эффективнее индекса.

### Index Scan

```sql
-- Идёт в индекс → получает указатель на строку → читает строку из heap
EXPLAIN SELECT * FROM users WHERE id = 42;

-- Index Scan using users_pkey on users  (cost=0.29..8.31 rows=1 width=64)
--   Index Cond: (id = 42)
```

### Index Only Scan

```sql
-- Все нужные данные есть в индексе — в heap ходить не нужно
-- Работает если visibility map актуален (после VACUUM)
EXPLAIN SELECT id FROM users WHERE id > 100 AND id < 200;

-- Index Only Scan using users_pkey on users  (cost=0.29..4.50 rows=99 width=8)
--   Index Cond: ((id > 100) AND (id < 200))
--   Heap Fetches: 0  -- 0 значит все данные из индекса
```

### Bitmap Index Scan + Bitmap Heap Scan

```sql
-- Двухфазный процесс:
-- 1) Bitmap Index Scan — собирает bitmap страниц, где есть совпадения
-- 2) Bitmap Heap Scan — читает эти страницы из heap
-- Эффективен для средней селективности (сотни-тысячи строк)

EXPLAIN SELECT * FROM orders WHERE status = 'pending';

-- Bitmap Heap Scan on orders  (cost=12.45..520.10 rows=2000 width=48)
--   Recheck Cond: (status = 'pending')
--   -> Bitmap Index Scan on idx_orders_status  (cost=0.00..12.00 rows=2000 width=0)
--        Index Cond: (status = 'pending')
```

## Типы соединений (Joins)

### Nested Loop

```sql
-- Для каждой строки внешней таблицы ищет совпадения во внутренней
-- Эффективен: малое количество строк во внешней таблице + индекс на внутренней
-- O(N * M), но с индексом O(N * log(M))

-- Nested Loop  (cost=0.29..41.65 rows=10 width=96)
--   -> Seq Scan on orders  (cost=0.00..1.10 rows=10 width=48)
--         Filter: (user_id = 42)
--   -> Index Scan using users_pkey on users  (cost=0.29..4.31 rows=1 width=48)
--         Index Cond: (id = orders.user_id)
```

### Hash Join

```sql
-- 1) Строит хэш-таблицу из меньшей таблицы
-- 2) Проходит по большей, ищет совпадения в хэш-таблице
-- Эффективен: обе таблицы достаточно большие, нет подходящего индекса
-- O(N + M) по времени, O(min(N,M)) по памяти

-- Hash Join  (cost=30.00..150.00 rows=5000 width=96)
--   Hash Cond: (orders.user_id = users.id)
--   -> Seq Scan on orders  (cost=0.00..80.00 rows=5000 width=48)
--   -> Hash  (cost=20.00..20.00 rows=1000 width=48)
--         -> Seq Scan on users  (cost=0.00..20.00 rows=1000 width=48)
```

### Merge Join

```sql
-- Обе стороны отсортированы — проходит параллельно по обеим
-- Эффективен: данные уже отсортированы (по индексу) или нужна сортировка по другим причинам
-- O(N*log(N) + M*log(M)) с сортировкой, O(N + M) без

-- Merge Join  (cost=0.56..200.00 rows=5000 width=96)
--   Merge Cond: (users.id = orders.user_id)
--   -> Index Scan using users_pkey on users  (cost=0.29..50.00 rows=1000 width=48)
--   -> Index Scan using idx_orders_user on orders  (cost=0.29..120.00 rows=5000 width=48)
```

## Типы индексов

### B-tree (default)

Стандартный индекс. Поддерживает операторы сравнения: `=`, `<`, `>`, `<=`, `>=`, `BETWEEN`, `IN`, `IS NULL`.

```sql
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_orders_created ON orders(created_at DESC);

-- Работает:
SELECT * FROM users WHERE email = 'alice@example.com';
SELECT * FROM orders WHERE created_at > '2024-01-01' ORDER BY created_at DESC;

-- НЕ работает с B-tree:
SELECT * FROM users WHERE email LIKE '%alice%';  -- LIKE с % в начале
```

### GIN (Generalized Inverted Index)

Для типов с множественными значениями: JSONB, массивы, полнотекстовый поиск.

```sql
-- JSONB
CREATE INDEX idx_items_meta ON items USING GIN (meta);

SELECT * FROM items WHERE meta @> '{"color": "red"}';  -- containment
SELECT * FROM items WHERE meta ? 'color';               -- key exists
SELECT * FROM items WHERE meta ?| array['color','size']; -- any key exists

-- Массивы
CREATE INDEX idx_posts_tags ON posts USING GIN (tags);

SELECT * FROM posts WHERE tags @> ARRAY['go', 'postgresql']; -- содержит оба
SELECT * FROM posts WHERE tags && ARRAY['go', 'rust'];       -- пересечение

-- Полнотекстовый поиск
CREATE INDEX idx_articles_fts ON articles USING GIN (to_tsvector('russian', title || ' ' || body));

SELECT * FROM articles
WHERE to_tsvector('russian', title || ' ' || body) @@ to_tsquery('russian', 'PostgreSQL & индексы');
```

### GiST (Generalized Search Tree)

Для геометрии, range types, полнотекстового поиска (альтернатива GIN).

```sql
-- Range types
CREATE INDEX idx_events_period ON events USING GIST (during);

SELECT * FROM events WHERE during && '[2024-01-01, 2024-02-01)'; -- пересечение
SELECT * FROM events WHERE during @> '2024-01-15'::date;         -- содержит

-- Полнотекстовый поиск (GiST vs GIN)
-- GiST: меньше размер индекса, медленнее поиск
-- GIN: больше размер, быстрее поиск, медленнее обновление
CREATE INDEX idx_articles_fts_gist ON articles USING GIST (tsv);
```

**GIN vs GiST для полнотекстового поиска**: GIN быстрее при чтении (в 3-10 раз), GiST быстрее при записи и занимает меньше места. Для read-heavy нагрузки — GIN, для write-heavy или небольших таблиц — GiST.

### BRIN (Block Range Index)

Для естественно упорядоченных данных. Хранит min/max значения для каждого блока страниц.

```sql
-- Идеален для timestamps, auto-increment (данные вставляются последовательно)
CREATE INDEX idx_logs_created ON logs USING BRIN (created_at);

-- Размер: в 100-1000 раз меньше B-tree!
-- Но: работает только если данные физически упорядочены на диске

SELECT * FROM logs WHERE created_at > NOW() - INTERVAL '1 hour';
```

**Когда BRIN**: append-only таблицы (логи, события), данные вставляются хронологически, таблица большая (миллионы строк).

## Продвинутые индексы

### Partial indexes

Индексирует только подмножество строк. Меньший размер, быстрее обновление.

```sql
-- Индекс только для неудалённых пользователей
CREATE INDEX idx_users_email_active ON users(email) WHERE deleted_at IS NULL;

-- Индекс только для необработанных заказов
CREATE INDEX idx_orders_pending ON orders(created_at) WHERE status = 'pending';

-- Запрос ДОЛЖЕН содержать условие из WHERE индекса, иначе индекс не используется
SELECT * FROM users WHERE email = 'alice@example.com' AND deleted_at IS NULL; -- использует
SELECT * FROM users WHERE email = 'alice@example.com';                         -- НЕ использует
```

### Expression indexes

```sql
-- Индекс на выражении
CREATE INDEX idx_users_lower_email ON users(lower(email));

-- Запрос ДОЛЖЕН использовать то же выражение
SELECT * FROM users WHERE lower(email) = 'alice@example.com'; -- использует
SELECT * FROM users WHERE email = 'alice@example.com';         -- НЕ использует
```

### Covering indexes (INCLUDE)

Добавление неключевых столбцов в индекс для Index Only Scan.

```sql
-- Без INCLUDE: Index Scan → нужно идти в heap за total и status
CREATE INDEX idx_orders_user ON orders(user_id);

-- С INCLUDE: Index Only Scan — всё из индекса
CREATE INDEX idx_orders_user_cover ON orders(user_id) INCLUDE (total, status);

-- Этот запрос будет Index Only Scan:
SELECT total, status FROM orders WHERE user_id = 42;
```

**Важно**: столбцы в INCLUDE не участвуют в поиске, только хранятся в листьях индекса.

### Composite indexes

Индекс на нескольких столбцах. Порядок имеет значение.

```sql
CREATE INDEX idx_orders_user_status ON orders(user_id, status);

-- Правило leftmost prefix: индекс используется только если запрос фильтрует по ЛЕВЫМ столбцам
SELECT * FROM orders WHERE user_id = 42;                          -- использует (prefix user_id)
SELECT * FROM orders WHERE user_id = 42 AND status = 'pending';   -- использует (оба столбца)
SELECT * FROM orders WHERE status = 'pending';                     -- НЕ использует (нет user_id)

-- Порядок сортировки в составном индексе тоже имеет значение
CREATE INDEX idx_orders_sort ON orders(user_id ASC, created_at DESC);

-- Эффективно:
SELECT * FROM orders WHERE user_id = 42 ORDER BY created_at DESC;
-- Неэффективно (обратный порядок):
SELECT * FROM orders WHERE user_id = 42 ORDER BY created_at ASC;
```

## Когда индексы вредят

1. **Write overhead**: каждый INSERT/UPDATE/DELETE обновляет все индексы таблицы
2. **Bloat**: после множества UPDATE/DELETE индекс "раздувается", нужен `REINDEX`
3. **Неиспользуемые индексы**: занимают место и замедляют запись без пользы
4. **Слишком много индексов**: таблица с 10+ индексами — INSERT может быть в разы медленнее

```sql
-- Мониторинг: найти неиспользуемые индексы
SELECT
    schemaname,
    relname AS table,
    indexrelname AS index,
    idx_scan AS times_used,
    pg_size_pretty(pg_relation_size(indexrelid)) AS size
FROM pg_stat_user_indexes
WHERE idx_scan = 0
  AND indexrelname NOT LIKE '%pkey%'
  AND indexrelname NOT LIKE '%unique%'
ORDER BY pg_relation_size(indexrelid) DESC;

-- Мониторинг: соотношение seq scan vs index scan
SELECT
    relname AS table,
    seq_scan,
    idx_scan,
    CASE WHEN seq_scan + idx_scan > 0
        THEN round(100.0 * idx_scan / (seq_scan + idx_scan), 1)
        ELSE 0
    END AS idx_scan_pct
FROM pg_stat_user_tables
ORDER BY seq_scan DESC;
```

## Поиск медленных запросов

```sql
-- Включить расширение pg_stat_statements
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Топ медленных запросов по суммарному времени
SELECT
    query,
    calls,
    round(total_exec_time::numeric, 2) AS total_ms,
    round(mean_exec_time::numeric, 2) AS avg_ms,
    rows
FROM pg_stat_statements
ORDER BY total_exec_time DESC
LIMIT 10;

-- Логирование медленных запросов в postgresql.conf
-- log_min_duration_statement = 100  -- логировать запросы дольше 100ms
```

## Использование в Go

```go
// EXPLAIN ANALYZE из Go (для отладки)
func explainQuery(ctx context.Context, pool *pgxpool.Pool, query string, args ...any) (string, error) {
    explainSQL := "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) " + query
    rows, err := pool.Query(ctx, explainSQL, args...)
    if err != nil {
        return "", err
    }
    defer rows.Close()

    var plan strings.Builder
    for rows.Next() {
        var line string
        if err := rows.Scan(&line); err != nil {
            return "", err
        }
        plan.WriteString(line)
        plan.WriteString("\n")
    }
    return plan.String(), nil
}
```

## Вопросы для собеседования

1. **Как найти медленные запросы в PostgreSQL?**
   Три основных способа: (1) `pg_stat_statements` — расширение, собирающее статистику по всем запросам: суммарное время, среднее время, количество вызовов; (2) `log_min_duration_statement` — логирование запросов, превышающих порог; (3) `EXPLAIN ANALYZE` — анализ конкретного запроса.

2. **Когда использовать GIN vs GiST?**
   GIN (Generalized Inverted Index) — для JSONB, массивов, полнотекстового поиска. Быстрее при чтении, но медленнее при записи и больше по размеру. GiST (Generalized Search Tree) — для геометрии, range types, и тоже для полнотекста. Быстрее при записи, компактнее, но медленнее при поиске. Для read-heavy нагрузки — GIN, для write-heavy — GiST.

3. **Что такое covering index и когда он полезен?**
   Covering index (INCLUDE) хранит дополнительные столбцы в листьях индекса. Позволяет выполнить Index Only Scan — все данные берутся из индекса без обращения к heap. Полезен когда запрос часто выбирает одни и те же столбцы по одному и тому же условию.

4. **Почему составной индекс (a, b) не ускоряет запрос WHERE b = ?**
   Из-за правила leftmost prefix: B-tree индекс упорядочен сначала по `a`, потом по `b`. Без условия на `a` PostgreSQL не может эффективно найти нужные записи по `b` — для этого нужен отдельный индекс на `b`.

5. **Когда BRIN лучше B-tree?**
   BRIN эффективен для больших таблиц с естественно упорядоченными данными (логи, события с timestamp). Он в 100-1000 раз меньше B-tree, но работает только если данные физически упорядочены на диске. Для случайного доступа или неупорядоченных данных BRIN бесполезен.

6. **Что означает большое расхождение между estimated и actual rows в EXPLAIN?**
   Планировщик использует устаревшую статистику. Нужно выполнить `ANALYZE table_name` для обновления статистики. Неточная оценка может привести к выбору неоптимального плана (например, Seq Scan вместо Index Scan).
