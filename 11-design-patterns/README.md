# 11. Паттерны проектирования

Классические GoF паттерны адаптированные для Go + Go-специфичные паттерны. Go — не ООП язык, поэтому многие паттерны выглядят иначе (проще).

## Содержание

### Порождающие (Creational)
1. [Singleton](creational/01-singleton.md) — через sync.Once
2. [Factory](creational/02-factory.md) — функции-конструкторы
3. [Builder](creational/03-builder.md) — поэтапное создание
4. [Object Pool](creational/04-object-pool.md) — sync.Pool

### Структурные (Structural)
1. [Adapter](structural/01-adapter.md) — приведение интерфейсов
2. [Decorator](structural/02-decorator.md) — обёртки, middleware
3. [Facade](structural/03-facade.md) — упрощение сложного API
4. [Composite](structural/04-composite.md) — древовидные структуры

### Поведенческие (Behavioral)
1. [Strategy](behavioral/01-strategy.md) — подмена алгоритма через интерфейс
2. [Observer](behavioral/02-observer.md) — pub/sub, events
3. [Chain of Responsibility](behavioral/03-chain-of-responsibility.md) — middleware цепочка
4. [Iterator](behavioral/04-iterator.md) — range-over-func (Go 1.23+)
5. [Command](behavioral/05-command.md) — инкапсуляция действий

### Go-специфичные
1. [Functional Options](go-specific/01-functional-options.md) — WithXxx паттерн
2. [Middleware](go-specific/02-middleware.md) — HTTP middleware
3. [Table-Driven Tests](go-specific/03-table-driven-tests.md) — идиоматичные тесты
4. [Dependency Injection](go-specific/04-dependency-injection.md) — через интерфейсы
