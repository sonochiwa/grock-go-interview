# Git Workflows

## Git Flow

```
main ────────●────────────────●──────────── (production)
             │                ↑
develop ──●──┼──●──●──●───●───┤──●── (integration)
          │  │     │     ↗    │
feature/  │  │     └────●     │
  login ──●──┘                │
                              │
release/  ────────────────────●
  1.0     (freeze, bugfix only)

Ветки:
  main     — production code (tagged releases)
  develop  — integration branch
  feature/ — новые фичи (от develop, в develop)
  release/ — подготовка релиза (от develop, в main + develop)
  hotfix/  — срочные фиксы (от main, в main + develop)

✅ Чёткая структура для команд с scheduled releases
❌ Сложный, много веток, медленный feedback loop
❌ Не подходит для continuous deployment
```

## GitHub Flow

```
main ──●──●──●──●──●──●── (always deployable)
       │     ↑  │     ↑
       └──●──┘  └──●──┘
       feature  feature

Правила:
  1. main ВСЕГДА deployable
  2. Branch от main для любой работы
  3. Commit и push регулярно
  4. Открыть PR когда ready (или Draft PR раньше)
  5. Code review → approve
  6. Merge → auto deploy

✅ Простой, быстрый
✅ Подходит для continuous deployment
✅ PR = discussion + review
❌ Нет staging/release concept (нужен feature flags)
```

## Trunk-Based Development

```
main ──●──●──●──●──●──●── (trunk, deploy from here)
       │  │  ↑  │     ↑
       │  └──┘  └──●──┘
       │ (1-2    short-lived
       │ commits) feature (< 1 day)
       │
       └──● feature flag: hidden behind toggle

Правила:
  1. Все коммитят в main (trunk) или short-lived branches (< 1 day)
  2. Feature flags для незавершённых фич
  3. CI/CD запускается на каждый commit в main
  4. Release branches ТОЛЬКО для hotfix (если нужны)

✅ Минимум merge conflicts
✅ Continuous integration в чистом виде
✅ Быстрый feedback loop
❌ Требует feature flags infrastructure
❌ Требует хороший CI/CD и тесты
❌ Дисциплина: маленькие коммиты

Google, Meta, Netflix используют trunk-based
```

## Сравнение

```
│ Аспект           │ Git Flow    │ GitHub Flow │ Trunk-Based  │
├──────────────────┼─────────────┼─────────────┼──────────────┤
│ Сложность        │ Высокая     │ Низкая      │ Низкая       │
│ Ветки            │ Много       │ main + feat │ main (trunk) │
│ Release cycle    │ Scheduled   │ Continuous  │ Continuous   │
│ Feature flags    │ Не нужны    │ Опционально │ Обязательно  │
│ Merge conflicts  │ Частые      │ Средние     │ Редкие       │
│ CI/CD            │ Сложный     │ Простой     │ Простой      │
│ Для кого         │ Enterprise  │ Стартапы    │ Большие ком. │
│                  │ mobile apps │ SaaS        │ Google/Meta  │
```

## Conventional Commits

```
Формат: <type>(<scope>): <description>

Types:
  feat:     новая функциональность
  fix:      исправление бага
  docs:     документация
  style:    форматирование (не влияет на код)
  refactor: рефакторинг (не feat, не fix)
  perf:     оптимизация производительности
  test:     добавление/исправление тестов
  chore:    build, CI, зависимости
  ci:       CI/CD конфигурация

Примеры:
  feat(auth): add JWT token refresh
  fix(api): handle nil pointer in user handler
  docs(readme): update installation instructions
  refactor(db): extract repository interface
  perf(cache): add Redis pipeline for batch operations
  test(auth): add integration tests for OAuth flow

Breaking change:
  feat(api)!: remove deprecated v1 endpoints
  # Или в теле:
  BREAKING CHANGE: /v1/* endpoints removed, use /v2/*

Зачем:
  - Автогенерация CHANGELOG
  - Semantic versioning (feat → minor, fix → patch, ! → major)
  - Понятная история
```
