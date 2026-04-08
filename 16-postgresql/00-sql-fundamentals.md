# SQL Fundamentals

Базовые знания SQL, которые спрашивают независимо от конкретной СУБД.

## Типы данных

| Категория | Типы | Заметки |
|-----------|------|---------|
| Числа | `INTEGER`, `BIGINT`, `NUMERIC`, `FLOAT` | `NUMERIC` — точный, `FLOAT` — приблизительный |
| Строки | `VARCHAR(n)`, `TEXT`, `CHAR(n)` | В PostgreSQL `TEXT` и `VARCHAR` без лимита одинаковы по производительности |
| Дата/время | `DATE`, `TIME`, `TIMESTAMP`, `TIMESTAMPTZ` | Всегда используй `TIMESTAMPTZ` для хранения времени |
| Булевы | `BOOLEAN` | `TRUE`, `FALSE`, `NULL` (three-valued logic) |
| JSON | `JSON`, `JSONB` | `JSONB` — бинарный, поддерживает индексы |
| UUID | `UUID` | `gen_random_uuid()` в PostgreSQL |

## SELECT и фильтрация

```sql
-- Базовый запрос
SELECT name, age FROM users WHERE age > 18 ORDER BY name LIMIT 10;

-- DISTINCT — уникальные значения
SELECT DISTINCT city FROM users;

-- IN, BETWEEN, LIKE, IS NULL
SELECT * FROM users WHERE city IN ('Moscow', 'SPb');
SELECT * FROM orders WHERE created_at BETWEEN '2025-01-01' AND '2025-12-31';
SELECT * FROM users WHERE name LIKE 'A%';        -- начинается на A
SELECT * FROM users WHERE deleted_at IS NULL;     -- именно IS NULL, не = NULL
```

## JOIN'ы

```
       INNER JOIN            LEFT JOIN             RIGHT JOIN            FULL JOIN
      ┌───┐ ┌───┐          ┌───┐ ┌───┐          ┌───┐ ┌───┐          ┌───┐ ┌───┐
      │ A │●│ B │          │███│●│ B │          │ A │●│███│          │███│●│███│
      └───┘ └───┘          └───┘ └───┘          └───┘ └───┘          └───┘ └───┘
     совпадения          все из A +            все из B +           все из обоих
                         совпадения            совпадения
```

```sql
-- INNER JOIN — только совпадения
SELECT u.name, o.total
FROM users u
INNER JOIN orders o ON u.id = o.user_id;

-- LEFT JOIN — все из левой + совпадения из правой (NULL если нет)
SELECT u.name, o.total
FROM users u
LEFT JOIN orders o ON u.id = o.user_id;

-- CROSS JOIN — декартово произведение (каждый с каждым)
SELECT * FROM sizes CROSS JOIN colors;

-- Self JOIN — таблица сама с собой (иерархии)
SELECT e.name, m.name AS manager
FROM employees e
LEFT JOIN employees m ON e.manager_id = m.id;
```

### Частый вопрос: чем LEFT JOIN отличается от INNER JOIN?

`LEFT JOIN` возвращает **все** строки из левой таблицы, даже если нет совпадения в правой (заполняет `NULL`). `INNER JOIN` вернёт только строки с совпадением в обеих таблицах.

## Агрегатные функции и GROUP BY

```sql
-- COUNT, SUM, AVG, MIN, MAX
SELECT city, COUNT(*) AS user_count, AVG(age) AS avg_age
FROM users
GROUP BY city
HAVING COUNT(*) > 10    -- фильтр ПОСЛЕ группировки
ORDER BY user_count DESC;
```

> **WHERE vs HAVING**: `WHERE` фильтрует строки до группировки, `HAVING` — после.

## Подзапросы

```sql
-- Скалярный подзапрос
SELECT name, (SELECT COUNT(*) FROM orders o WHERE o.user_id = u.id) AS order_count
FROM users u;

-- EXISTS — проверка существования (часто быстрее IN)
SELECT * FROM users u
WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id);

-- IN с подзапросом
SELECT * FROM users WHERE id IN (SELECT user_id FROM orders WHERE total > 1000);
```

## Оконные функции

Оконные функции выполняют вычисление по набору строк, **не схлопывая** их (в отличие от `GROUP BY`).

```sql
-- ROW_NUMBER — порядковый номер в окне
SELECT name, salary,
       ROW_NUMBER() OVER (ORDER BY salary DESC) AS rank
FROM employees;

-- RANK vs DENSE_RANK
-- RANK:       1, 2, 2, 4  (пропускает)
-- DENSE_RANK: 1, 2, 2, 3  (не пропускает)

-- PARTITION BY — окно внутри группы
SELECT department, name, salary,
       RANK() OVER (PARTITION BY department ORDER BY salary DESC) AS dept_rank
FROM employees;

-- LAG / LEAD — предыдущая / следующая строка
SELECT date, revenue,
       revenue - LAG(revenue) OVER (ORDER BY date) AS diff
FROM daily_sales;

-- Накопительная сумма
SELECT date, revenue,
       SUM(revenue) OVER (ORDER BY date) AS running_total
FROM daily_sales;
```

## Нормализация

| Форма | Правило | Пример нарушения |
|-------|---------|------------------|
| **1NF** | Атомарные значения, нет повторяющихся групп | `phones: "123,456"` в одном поле |
| **2NF** | 1NF + нет частичных зависимостей от составного ключа | Имя студента зависит от student_id, но PK = (student_id, course_id) |
| **3NF** | 2NF + нет транзитивных зависимостей | `city` → `country` через `user.city` |

> На практике обычно стремятся к 3NF, но **денормализуют** ради производительности (read-heavy сценарии).

## ACID

| Свойство | Что значит | Пример |
|----------|-----------|--------|
| **Atomicity** | Транзакция выполняется целиком или откатывается | Перевод денег: списание + зачисление |
| **Consistency** | БД переходит из одного валидного состояния в другое | Constraints, triggers не нарушены |
| **Isolation** | Параллельные транзакции не мешают друг другу | Уровни: READ COMMITTED, REPEATABLE READ, SERIALIZABLE |
| **Durability** | После COMMIT данные сохранены даже при сбое | WAL (Write-Ahead Log) |

## Уровни изоляции

| Уровень | Dirty Read | Non-Repeatable Read | Phantom Read |
|---------|-----------|-------------------|-------------|
| READ UNCOMMITTED | ✅ | ✅ | ✅ |
| READ COMMITTED | ❌ | ✅ | ✅ |
| REPEATABLE READ | ❌ | ❌ | ✅* |
| SERIALIZABLE | ❌ | ❌ | ❌ |

> *В PostgreSQL REPEATABLE READ также защищает от phantom reads (реализация через MVCC/SSI).

PostgreSQL по умолчанию: **READ COMMITTED**.

## Частые вопросы на собеседовании

1. **В чём разница между `DELETE` и `TRUNCATE`?**
   - `DELETE` — DML, можно с WHERE, медленнее, логирует каждую строку, можно откатить
   - `TRUNCATE` — DDL, удаляет всё, быстрее, сбрасывает auto-increment

2. **Что такое `NULL`?**
   - NULL ≠ 0, NULL ≠ ''. Это отсутствие значения. `NULL = NULL` → `NULL` (не TRUE). Используй `IS NULL`.

3. **Что такое индекс?**
   - Структура данных (обычно B-tree) для ускорения поиска. Подробнее в [06-indexes-and-explain.md](06-indexes-and-explain.md).

4. **Что такое `EXPLAIN`?**
   - Показывает план выполнения запроса. Подробнее в [06-indexes-and-explain.md](06-indexes-and-explain.md).
