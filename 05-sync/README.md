# 05. Примитивы синхронизации

Когда каналов недостаточно или неудобно — используем sync пакет. Mutex, WaitGroup, Once, Pool, atomic и расширения из x/sync.

## Каналы vs Mutex

| Каналы | Mutex |
|---|---|
| Передача данных между горутинами | Защита shared state |
| Сигнализация (done, cancel) | Критическая секция |
| Pipeline, fan-out/fan-in | Кеш, счётчики, конфиг |

## Содержание

1. [Mutex](01-mutex.md) — взаимное исключение, deadlocks
2. [RWMutex](02-rwmutex.md) — много читателей, мало писателей
3. [WaitGroup](03-waitgroup.md) — ожидание группы горутин
4. [Once](04-once.md) — однократная инициализация
5. [Pool](05-pool.md) — переиспользование объектов
6. [Cond](06-cond.md) — условные переменные
7. [Atomic](07-atomic.md) — атомарные операции
8. [Errgroup](08-errgroup.md) — параллельные задачи с ошибками
9. [Singleflight](09-singleflight.md) — дедупликация вызовов
