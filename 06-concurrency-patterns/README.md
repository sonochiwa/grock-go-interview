# 06. Паттерны конкурентности

Готовые рецепты для конкурентных задач. Каждый паттерн решает конкретную проблему — знай, когда какой применять.

## Содержание

1. [Pipeline](01-pipeline.md) — цепочка стадий обработки
2. [Fan-Out / Fan-In](02-fan-out-fan-in.md) — распределение и сбор работы
3. [Worker Pool](03-worker-pool.md) — пул воркеров
4. [Semaphore](04-semaphore.md) — ограничение параллелизма
5. [Rate Limiting](05-rate-limiting.md) — ограничение скорости
6. [Or-Channel](06-or-channel.md) — первый результат
7. [Context Patterns](07-context-patterns.md) — каскадная отмена
8. [Pub/Sub](08-pub-sub.md) — издатель-подписчик
9. [Barrier](09-barrier.md) — барьерная синхронизация

## Визуальная карта

```
Pipeline:     [Gen] → [Stage1] → [Stage2] → [Stage3]

Fan-Out:      [Source] → [Worker1]
                      → [Worker2]  → [Merge] → [Output]
                      → [Worker3]

Worker Pool:  [Jobs] → [Pool of N Workers] → [Results]

Pub/Sub:      [Publisher] → [Topic] → [Sub1]
                                    → [Sub2]
                                    → [Sub3]
```


---

## Задачи

Практические задачи по этой теме: [exercises/](exercises/)
