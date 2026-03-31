# Git Hooks

## Типы хуков

```
Client-side (локальные):
  pre-commit      — перед commit (lint, format, tests)
  prepare-commit-msg — подготовка commit message
  commit-msg      — валидация commit message
  post-commit     — после commit (notification)
  pre-push        — перед push (тесты!)
  pre-rebase      — перед rebase
  post-checkout   — после checkout/switch
  post-merge      — после merge

Server-side (на remote):
  pre-receive     — перед принятием push
  update          — для каждой ветки в push
  post-receive    — после принятием push (CI trigger, notification)

Расположение: .git/hooks/ (локальные, не коммитятся!)
Для sharing: .githooks/ + git config core.hooksPath .githooks
```

## Примеры хуков

### pre-commit

```bash
#!/bin/bash
# .githooks/pre-commit
set -e

echo "Running pre-commit checks..."

# Go formatting
UNFORMATTED=$(gofmt -l .)
if [[ -n "$UNFORMATTED" ]]; then
    echo "❌ Unformatted Go files:"
    echo "$UNFORMATTED"
    echo "Run: gofmt -w ."
    exit 1
fi

# Go vet
go vet ./...

# Lint
golangci-lint run ./...

# Проверка на secrets
if git diff --cached --diff-filter=ACM | grep -qE '(password|secret|api_key)\s*=\s*"[^"]+"|AKIA[0-9A-Z]{16}'; then
    echo "❌ Possible secrets detected in staged files!"
    exit 1
fi

echo "✅ Pre-commit checks passed"
```

### commit-msg (conventional commits)

```bash
#!/bin/bash
# .githooks/commit-msg
MSG=$(cat "$1")
PATTERN="^(feat|fix|docs|style|refactor|perf|test|chore|ci)(\(.+\))?!?: .{1,72}$"

if ! echo "$MSG" | head -1 | grep -qE "$PATTERN"; then
    echo "❌ Invalid commit message format!"
    echo "Expected: <type>(<scope>): <description>"
    echo "Types: feat, fix, docs, style, refactor, perf, test, chore, ci"
    echo ""
    echo "Examples:"
    echo "  feat(auth): add JWT refresh token"
    echo "  fix(api): handle nil pointer"
    exit 1
fi
```

### pre-push

```bash
#!/bin/bash
# .githooks/pre-push
set -e

echo "Running pre-push checks..."

# Запретить push в main напрямую
BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [[ "$BRANCH" == "main" || "$BRANCH" == "master" ]]; then
    echo "❌ Direct push to $BRANCH is not allowed!"
    echo "Create a feature branch and open a PR."
    exit 1
fi

# Тесты
go test -race -count=1 ./...

echo "✅ Pre-push checks passed"
```

## Управление хуками

```bash
# Настройка для команды
git config core.hooksPath .githooks
# Теперь хуки из .githooks/ (коммитятся в репо!)

# Makefile
.PHONY: setup
setup:
	git config core.hooksPath .githooks
	chmod +x .githooks/*

# Пропустить хук (крайний случай!)
git commit --no-verify -m "urgent hotfix"
git push --no-verify
```

## Pre-commit Framework

```yaml
# .pre-commit-config.yaml (https://pre-commit.com)
repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.5.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-yaml
      - id: check-added-large-files
        args: ['--maxkb=500']

  - repo: https://github.com/golangci/golangci-lint
    rev: v1.57.0
    hooks:
      - id: golangci-lint

  - repo: https://github.com/commitizen-tools/commitizen
    rev: v3.15.0
    hooks:
      - id: commitizen
        stages: [commit-msg]
```

```bash
pip install pre-commit
pre-commit install           # setup hooks
pre-commit run --all-files   # run manually
```

## CI Integration

```
Хуки НЕ заменяют CI!

Хуки:
  ✅ Быстрый feedback (до push)
  ✅ Ловят простые ошибки (formatting, lint)
  ❌ Можно пропустить (--no-verify)
  ❌ Не стандартизированы (зависит от machine)

CI (GitHub Actions):
  ✅ Обязательный (нельзя пропустить)
  ✅ Одинаковое окружение
  ✅ Integration tests, build, deploy
  ❌ Медленнее (push → CI → результат)

Стратегия:
  pre-commit: fast checks (format, vet, lint)
  pre-push: unit tests
  CI: full test suite, integration tests, build, security scan
```
