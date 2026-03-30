# Green Tea GC (Go 1.26)

## Обзор

Green Tea — новый generational GC, включённый по умолчанию в Go 1.26. Основная идея: большинство объектов умирает молодыми (generational hypothesis). Собирать молодые объекты чаще и дешевле.

## Отличия от классического GC

| Classic (до 1.26) | Green Tea (1.26+) |
|---|---|
| Одно поколение — сканируем ВСЁ | Два поколения: young + old |
| Full heap scan каждый цикл | Minor GC: только young generation |
| GOGC контролирует частоту | GOGC + young gen sizing |
| ~25% CPU на GC | Меньше CPU на типичных нагрузках |
| Sub-ms STW | Ещё меньше STW (меньше работы) |

## Как работает

```
Аллокация → Young Generation (nursery)
                │
                ▼ (minor GC — частый, быстрый)
         Выжившие объекты
                │
                ▼ (promotion)
         Old Generation
                │
                ▼ (major GC — редкий, дороже)
         Полная сборка
```

### Minor GC (молодое поколение)

1. Сканируем только young generation + remembered set
2. Живые объекты → promote в old generation
3. Мёртвые → освободить
4. Очень быстро: young gen маленький, большинство объектов уже мертво

### Major GC (старое поколение)

- Полная сборка обоих поколений
- Запускается реже (когда old gen вырос)
- Аналогична классическому tri-color mark-sweep

### Remembered Set (card table)

Проблема: old object может указывать на young object. Без tracked ссылок пропустим живой young объект.

```
Old Gen → Young Gen (inter-generational pointer)
         │
         └─ Write barrier записывает в card table
            │
            └─ Minor GC проверяет card table как дополнительные roots
```

## Что изменилось для разработчика

- **Ничего в коде менять не надо** — полностью прозрачно
- **Меньше CPU на GC** для типичных нагрузок (много короткоживущих объектов)
- **GOGC** всё ещё работает, но семантика немного другая
- **GOMEMLIMIT** всё ещё работает
- **Отключение**: `GOEXPERIMENT=nogreenteagc` или `GOGC` настройка

## Выигрыш

- Приложения с большим heap (>1GB) — значительно меньше CPU на GC
- Приложения с высокой аллокацией — minor GC дешевле full GC
- Latency-sensitive сервисы — меньше и реже STW паузы

## Частые вопросы на собеседованиях

**Q: Зачем Go добавил generational GC?**
A: Generational hypothesis — большинство объектов умирает молодыми. Собирать только молодые = меньше работы = меньше CPU = меньше паузы.

**Q: Почему раньше не добавляли?**
A: Write barrier для поколений был дорогой. Green Tea оптимизировал card table и write barrier для Go-специфичных паттернов.

**Q: Нужно ли менять код для Green Tea?**
A: Нет. Полностью прозрачно. Но можно тюнить GOGC/GOMEMLIMIT.
