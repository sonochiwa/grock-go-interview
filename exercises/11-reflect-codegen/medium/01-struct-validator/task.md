# Struct Validator

Напиши `Validate(v any) []ValidationError` используя reflect.

Поддержи теги:
- `validate:"required"` — поле не должно быть zero value
- `validate:"min=N"` — для int/string(len): >= N
- `validate:"max=N"` — для int/string(len): <= N

`ValidationError{Field string, Tag string, Message string}`
