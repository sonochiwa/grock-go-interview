# PGO (Profile-Guided Optimization)

## Обзор

PGO (Go 1.20+, GA в 1.21) — компилятор использует production CPU profile для оптимизации: inlining горячих функций, devirtualization, оптимизация branch prediction.

## Как использовать

```bash
# 1. Собрать профиль с продакшена
curl -o default.pgo http://myapp:6060/debug/pprof/profile?seconds=30

# 2. Положить default.pgo в корень main пакета
cp default.pgo ./cmd/myapp/default.pgo

# 3. Собрать с PGO (автоматически если default.pgo найден)
go build ./cmd/myapp

# Или явно
go build -pgo=./cpu.pprof ./cmd/myapp
```

## Что оптимизирует PGO

- **Inlining**: горячие функции инлайнятся агрессивнее
- **Devirtualization**: если interface метод вызывается всегда для одного типа → прямой вызов
- **Оптимизация ветвлений**: hot paths предсказываются лучше

Типичный выигрыш: **2-7% CPU**.

## Частые вопросы на собеседованиях

**Q: Что такое PGO?**
A: Компилятор использует CPU профиль для оптимизации: агрессивнее инлайнит горячие функции, девиртуализирует интерфейсные вызовы.

**Q: Откуда брать профиль?**
A: С production или staging. Именно реальная нагрузка показывает горячие пути. Файл `default.pgo` в корне main пакета подхватывается автоматически.
