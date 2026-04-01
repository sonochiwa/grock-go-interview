# Error Wrapping

Реализуй 3-уровневую обработку ошибок:

1. **Repository**: `GetUser(id int) (User, error)` — возвращает `ErrNotFound` если id не найден
2. **Service**: `GetUserService(id int) (User, error)` — оборачивает ошибку repository через `%w`
3. **Handler**: `HandleGetUser(id int) (User, int, error)` — возвращает user, HTTP status code, error
   - `ErrNotFound` → 404
   - Другая ошибка → 500
   - Успех → 200

Проверь, что `errors.Is(err, ErrNotFound)` работает через всю цепочку.
