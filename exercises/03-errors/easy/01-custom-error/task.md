# Custom Error

Создай тип `ValidationError` с полями `Field` и `Message` (оба string).
Реализуй интерфейс `error`.

Напиши функцию `ValidateAge(age int) error`:
- age < 0 → ValidationError{Field: "age", Message: "must be non-negative"}
- age > 150 → ValidationError{Field: "age", Message: "must be <= 150"}
- иначе → nil

Убедись, что ошибку можно извлечь через `errors.As`.
