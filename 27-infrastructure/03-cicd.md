# CI/CD для Go (GitLab)

## Linting

```yaml
# .golangci.yml
linters:
  enable:
    - errcheck       # проверка необработанных ошибок
    - govet          # go vet
    - staticcheck    # мощный статический анализ
    - gosimple       # упрощение кода
    - ineffassign    # неиспользуемые присваивания
    - unused         # неиспользуемый код
    - gocritic       # стилистические проверки
    - revive         # замена golint
    - gosec          # security issues
    - prealloc       # предаллокация слайсов
    - noctx          # HTTP requests без context

linters-settings:
  govet:
    enable-all: true
  errcheck:
    check-type-assertions: true
  gocritic:
    enabled-tags:
      - performance
      - diagnostic

issues:
  exclude-use-default: false
  max-issues-per-linter: 0
  max-same-issues: 0
```

```bash
# Установка
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Запуск
golangci-lint run ./...
```

## GitLab CI/CD

### Базовый .gitlab-ci.yml

GitLab CI конфигурируется через файл `.gitlab-ci.yml` в корне репозитория. Пайплайн состоит из **stages** -- этапы выполняются последовательно, джобы внутри одного этапа -- параллельно.

```yaml
# .gitlab-ci.yml

# Базовый образ для всех джобов (можно переопределить в каждом)
image: golang:1.24-alpine

# Этапы пайплайна (порядок важен)
stages:
  - lint
  - test
  - build
  - deploy

# Глобальные переменные
variables:
  GOPATH: $CI_PROJECT_DIR/.go
  GOFLAGS: "-mod=readonly"

# Кэширование go modules -- переиспользуется между пайплайнами
cache:
  key:
    files:
      - go.sum
  paths:
    - .go/pkg/mod/
  policy: pull-push
```

**Ключевые концепции:**
- `image` -- Docker-образ, в котором выполняется джоб
- `stages` -- порядок этапов; если lint упал, test/build/deploy не запустятся
- `cache` -- сохраняется между запусками пайплайна (go modules, бинарники линтера)
- `artifacts` -- передаются между джобами внутри одного пайплайна (бинарники, отчёты)
- `key: files: [go.sum]` -- кэш инвалидируется при изменении зависимостей

### Lint stage

```yaml
lint:
  stage: lint
  image: golangci/golangci-lint:v2.1-alpine
  script:
    - golangci-lint run --timeout=5m ./...
  # Кэш не нужен -- образ уже содержит линтер
  cache: []
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH
```

Используем официальный образ `golangci/golangci-lint` -- в нём уже установлен линтер нужной версии. Конфигурация берётся из `.golangci.yml` в корне проекта.

### Test stage

#### Unit-тесты с race detector

```yaml
unit-tests:
  stage: test
  script:
    - go test -race -count=1 -coverprofile=coverage.out ./...
    - go tool cover -func=coverage.out
    - go tool cover -html=coverage.out -o coverage.html
  artifacts:
    reports:
      # GitLab парсит Cobertura-формат и показывает покрытие в MR diff
      coverage_report:
        coverage_format: cobertura
        path: coverage.xml
    paths:
      - coverage.html
    expire_in: 7 days
  coverage: '/total:\s+\(statements\)\s+(\d+\.\d+)%/'
  before_script:
    # gocover-cobertura конвертирует Go coverage в Cobertura XML
    - go install github.com/boumenot/gocover-cobertura@latest
  after_script:
    - gocover-cobertura < coverage.out > coverage.xml
```

**Coverage badge** -- регулярное выражение в поле `coverage` парсит вывод `go tool cover -func` и GitLab автоматически показывает badge с процентом покрытия. Настроить badge можно в Settings > CI/CD > General pipelines > Test coverage parsing.

#### Integration-тесты с services

GitLab CI позволяет запускать сервисные контейнеры (PostgreSQL, Redis и др.) рядом с джобом. Сервисы доступны по hostname, совпадающему с именем образа (слэши и двоеточия заменяются на дефисы).

```yaml
integration-tests:
  stage: test
  services:
    - name: postgres:16-alpine
      alias: postgres
      variables:
        POSTGRES_USER: test
        POSTGRES_PASSWORD: test
        POSTGRES_DB: testdb
    - name: redis:7-alpine
      alias: redis
  variables:
    DATABASE_URL: "postgres://test:test@postgres:5432/testdb?sslmode=disable"
    REDIS_URL: "redis://redis:6379"
  script:
    - |
      # Ждём готовности PostgreSQL
      apk add --no-cache postgresql-client
      until pg_isready -h postgres -U test; do sleep 1; done
    - go test -race -tags integration -count=1 -timeout=5m ./...
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH
```

**Важно:** в GitLab CI сервисы доступны по `alias` (или по имени образа), а не по `localhost`. Это отличие от локальной разработки -- используйте переменные окружения для URL-ов.

### Build stage

#### Сборка бинарника

```yaml
build:
  stage: build
  script:
    - |
      CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
        go build \
        -ldflags="-s -w -X main.version=${CI_COMMIT_TAG:-$CI_COMMIT_SHORT_SHA}" \
        -o bin/server ./cmd/server
  artifacts:
    paths:
      - bin/server
    expire_in: 1 week
  rules:
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH
    - if: $CI_COMMIT_TAG
```

- `CGO_ENABLED=0` -- статическая линковка, бинарник работает в scratch/distroless
- `-ldflags="-s -w"` -- убираем таблицу символов и DWARF (~30% меньше бинарник)
- `-X main.version=...` -- вшиваем версию на этапе сборки

#### Docker build с Kaniko

Docker-in-Docker (dind) требует privileged mode, что создаёт проблемы с безопасностью. **Kaniko** собирает образы в userspace без Docker daemon.

```yaml
docker-build:
  stage: build
  image:
    name: gcr.io/kaniko-project/executor:v1.23.2-debug
    entrypoint: [""]
  script:
    - |
      /kaniko/executor \
        --context $CI_PROJECT_DIR \
        --dockerfile $CI_PROJECT_DIR/Dockerfile \
        --destination $CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA \
        --destination $CI_REGISTRY_IMAGE:latest \
        --cache=true \
        --cache-repo=$CI_REGISTRY_IMAGE/cache
  rules:
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH
    - if: $CI_COMMIT_TAG
  before_script:
    # Авторизация в GitLab Container Registry
    - mkdir -p /kaniko/.docker
    - |
      echo "{\"auths\":{\"$CI_REGISTRY\":{\"auth\":\"$(echo -n ${CI_REGISTRY_USER}:${CI_REGISTRY_PASSWORD} | base64)\"}}}" \
        > /kaniko/.docker/config.json
```

**Преимущества Kaniko:**
- Не требует privileged mode
- Встроенное кэширование слоёв (`--cache=true`)
- Работает в любом Kubernetes-кластере
- `$CI_REGISTRY_*` -- предопределённые переменные GitLab для Container Registry

### Deploy stage

```yaml
deploy-staging:
  stage: deploy
  image: alpine:latest
  environment:
    name: staging
    url: https://staging.myapp.example.com
  before_script:
    - apk add --no-cache openssh-client
    - eval $(ssh-agent -s)
    - echo "$SSH_PRIVATE_KEY" | ssh-add -
  script:
    - ssh $STAGING_USER@$STAGING_HOST "docker pull $CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA"
    - ssh $STAGING_USER@$STAGING_HOST "docker compose -f /app/docker-compose.yml up -d"
  rules:
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH

deploy-production:
  stage: deploy
  image: alpine:latest
  environment:
    name: production
    url: https://myapp.example.com
  before_script:
    - apk add --no-cache openssh-client
    - eval $(ssh-agent -s)
    - echo "$SSH_PRIVATE_KEY" | ssh-add -
  script:
    - ssh $PROD_USER@$PROD_HOST "docker pull $CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA"
    - ssh $PROD_USER@$PROD_HOST "docker compose -f /app/docker-compose.yml up -d"
  rules:
    - if: $CI_COMMIT_TAG =~ /^v\d+\.\d+\.\d+$/
  when: manual
  allow_failure: false
```

- `environment` -- GitLab отслеживает деплои и показывает их в разделе Environments
- `when: manual` -- деплой на прод требует ручного подтверждения в интерфейсе
- `allow_failure: false` -- пайплайн блокируется до ручного подтверждения
- SSH-ключи хранятся в Settings > CI/CD > Variables (тип File или masked)

### GitLab Container Registry

GitLab имеет встроенный Docker Registry для каждого проекта. Предопределённые переменные:

| Переменная | Значение |
|---|---|
| `$CI_REGISTRY` | `registry.gitlab.com` |
| `$CI_REGISTRY_IMAGE` | `registry.gitlab.com/group/project` |
| `$CI_REGISTRY_USER` | `gitlab-ci-token` |
| `$CI_REGISTRY_PASSWORD` | автоматический токен джоба |

```yaml
# Пример: сборка и пуш с тегами
docker-push:
  stage: build
  image:
    name: gcr.io/kaniko-project/executor:v1.23.2-debug
    entrypoint: [""]
  before_script:
    - mkdir -p /kaniko/.docker
    - |
      echo "{\"auths\":{\"$CI_REGISTRY\":{\"auth\":\"$(echo -n ${CI_REGISTRY_USER}:${CI_REGISTRY_PASSWORD} | base64)\"}}}" \
        > /kaniko/.docker/config.json
  script:
    - |
      # Для тегов -- пушим с версией, для main -- с SHA и latest
      if [ -n "$CI_COMMIT_TAG" ]; then
        TAGS="--destination $CI_REGISTRY_IMAGE:$CI_COMMIT_TAG"
      else
        TAGS="--destination $CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA --destination $CI_REGISTRY_IMAGE:latest"
      fi
      /kaniko/executor \
        --context $CI_PROJECT_DIR \
        --dockerfile $CI_PROJECT_DIR/Dockerfile \
        $TAGS \
        --cache=true
  rules:
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH
    - if: $CI_COMMIT_TAG
```

Образы доступны через: `docker pull registry.gitlab.com/your-group/your-project:tag`

Cleanup policy настраивается в Settings > Packages & Registries > Container Registry -- автоудаление старых образов.

### Advanced

#### Merge Request Pipelines

По умолчанию пайплайн запускается на каждый пуш в любую ветку. Для экономии ресурсов используйте `rules` с `merge_request_event`:

```yaml
# Запускать только для MR и default branch
workflow:
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH
    - if: $CI_COMMIT_TAG
```

Это предотвращает дублирование пайплайнов (один от пуша, другой от MR).

#### rules vs only/except

`only/except` -- устаревший синтаксис. Используйте `rules`:

```yaml
# Плохо (устаревшее)
deploy:
  only:
    - main
  except:
    - schedules

# Хорошо
deploy:
  rules:
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH && $CI_PIPELINE_SOURCE != "schedule"
```

#### Variables

```yaml
# Глобальные переменные
variables:
  GO_VERSION: "1.24"
  APP_NAME: "myapp"

# Переопределение в джобе
test:
  variables:
    GOFLAGS: "-race"
```

Секреты (пароли, токены) хранятся в Settings > CI/CD > Variables:
- **Masked** -- не отображается в логах
- **Protected** -- доступна только в protected branches/tags
- **File** -- записывается во временный файл, переменная содержит путь

#### Cache vs Artifacts

| | Cache | Artifacts |
|---|---|---|
| Назначение | Ускорение пайплайнов | Передача файлов между джобами |
| Хранение | Между пайплайнами | Внутри одного пайплайна |
| Гарантия | Best-effort (может не быть) | Гарантированная доставка |
| Пример | go modules, бинарники линтера | собранный бинарник, coverage report |

```yaml
# Cache -- go modules
cache:
  key:
    files: [go.sum]
  paths:
    - .go/pkg/mod/

# Artifacts -- передача бинарника из build в deploy
build:
  artifacts:
    paths:
      - bin/server
    expire_in: 1 day
```

#### Include и шаблоны

Вынесите общие конфигурации в переиспользуемые шаблоны:

```yaml
# .gitlab/ci/go.yml -- шаблон для Go-проектов
.go-base:
  image: golang:${GO_VERSION}-alpine
  cache:
    key:
      files: [go.sum]
    paths:
      - .go/pkg/mod/

# .gitlab-ci.yml
include:
  - local: '.gitlab/ci/go.yml'
  # Шаблоны из другого проекта
  - project: 'devops/ci-templates'
    ref: main
    file: '/templates/go-pipeline.yml'
  # Удалённый шаблон
  - remote: 'https://example.com/templates/security.yml'

lint:
  extends: .go-base
  stage: lint
  script:
    - golangci-lint run ./...
```

#### Multi-project pipelines

Триггер пайплайна в другом проекте (например, деплой инфраструктуры после сборки):

```yaml
trigger-deploy:
  stage: deploy
  trigger:
    project: devops/infrastructure
    branch: main
    strategy: depend   # ждать завершения дочернего пайплайна
  variables:
    APP_VERSION: $CI_COMMIT_SHORT_SHA
```

#### Parallel jobs

```yaml
test:
  stage: test
  parallel:
    matrix:
      - GO_VERSION: ["1.23", "1.24"]
        DB_VERSION: ["postgres:15-alpine", "postgres:16-alpine"]
  image: golang:${GO_VERSION}-alpine
  services:
    - name: ${DB_VERSION}
      alias: postgres
  script:
    - go test -race ./...
```

Это создаст 4 джоба (2 версии Go x 2 версии PostgreSQL), запущенных параллельно. Удобно для матрицы совместимости.

## Makefile

```makefile
.PHONY: all build test lint run clean

BINARY=server
VERSION=$(shell git describe --tags --always)
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

all: lint test build

build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/server

test:
	go test -race -count=1 ./...

test-integration:
	go test -race -tags integration -count=1 ./...

test-cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

run:
	go run ./cmd/server

generate:
	go generate ./...
	buf generate

migrate-up:
	goose -dir migrations postgres "$$DATABASE_URL" up

migrate-down:
	goose -dir migrations postgres "$$DATABASE_URL" down

docker-build:
	docker build -t myapp:$(VERSION) .

clean:
	rm -rf bin/ coverage.out coverage.html
```

## Vulnerability Scanning

```bash
# Встроенный в Go
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# В CI:
- name: Vulnerability check
  run: govulncheck ./...

# Docker image scanning
docker scout cves myapp:latest
# или
trivy image myapp:latest
```
