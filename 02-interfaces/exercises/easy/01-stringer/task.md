# Stringer

Создай тип `Money` с полями `Amount` (float64) и `Currency` (string).

Реализуй интерфейс `fmt.Stringer`, чтобы:
- `Money{100.50, "USD"}` → `"100.50 USD"`
- `Money{0, "EUR"}` → `"0.00 EUR"`

Формат: 2 знака после точки + пробел + валюта.
