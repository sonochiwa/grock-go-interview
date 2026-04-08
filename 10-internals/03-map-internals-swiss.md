# Map Internals: Swiss Table (Go 1.24+)

## Обзор

С Go 1.24 map по умолчанию использует Swiss Table — хеш-таблицу с open addressing. Быстрее классической на 20-50% для lookup.

## Отличия от classic

| Classic (до 1.24) | Swiss Table (1.24+) |
|---|---|
| Separate chaining (overflow buckets) | Open addressing (probing) |
| Bucket = 8 слотов | Group = 16 слотов |
| tophash[8]uint8 | Control bytes: metadata для SIMD |
| Overflow → linked list | Probing → следующая группа |
| Load factor 6.5 | ~87.5% (7/8) |

## Структура

```
Swiss Table:
┌─────────────────────────────────────────────────────┐
│ Control bytes (16 байт):                             │
│ [ctrl0][ctrl1]...[ctrl15]                            │
│  каждый ctrl: 7 бит hash + 1 бит (empty/deleted/full)│
├─────────────────────────────────────────────────────┤
│ Slots (16 пар key-value):                            │
│ [k0,v0] [k1,v1] ... [k15,v15]                      │
└─────────────────────────────────────────────────────┘

Group = 16 control bytes + 16 slots
```

## Как работает lookup

1. Вычислить хеш: `H1 = hash >> 7` (номер группы), `H2 = hash & 0x7F` (7-бит tag)
2. Загрузить 16 control bytes группы
3. **SIMD**: сравнить H2 со всеми 16 control bytes за ОДНУ инструкцию (SSE2 на x86)
4. Получить bitmask совпадений
5. Проверить полные ключи для совпавших позиций
6. Если не нашли — quadratic probing → следующая группа

### Почему быстрее

- **SIMD**: 16 сравнений за одну CPU инструкцию вместо последовательного перебора
- **Cache-friendly**: control bytes — 16 байт (одна cache line)
- **Меньше pointer chasing**: нет overflow buckets (linked list)
- **На платформах без SIMD**: побитовые хаки для параллельного сравнения

## Инкрементальный рост

Swiss Table в Go также поддерживает инкрементальную эвакуацию (как classic). Рост при достижении load factor ~87.5%.

## Частые вопросы на собеседованиях

**Q: Что такое Swiss Table?**
A: Hash table с open addressing, использующая SIMD для параллельного поиска в группах по 16 слотов.

**Q: Почему Go перешёл на Swiss Table?**
A: Быстрее для lookup (основная операция), лучше cache locality, меньше аллокаций (нет overflow buckets).

**Q: Изменился ли API map?**
A: Нет. Это чистая implementation detail. Код не нужно менять.

**Q: С какой версии Swiss Table по умолчанию?**
A: Go 1.24. В 1.23 был доступен как эксперимент (GOEXPERIMENT=swissmap).
