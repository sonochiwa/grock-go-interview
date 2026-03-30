# Mock Repository

Дан интерфейс `UserRepository` и `UserService`.

1. Реализуй `MockUserRepository` для тестов (записывает вызовы)
2. Напиши тесты для `UserService.GetByID` и `UserService.Create`:
   - User found / not found
   - Create success / duplicate email
   - Validation error (empty name)
