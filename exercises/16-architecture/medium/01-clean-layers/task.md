# Clean Architecture Layers

Реализуй CRUD для сущности Task в Clean Architecture:

**Domain**: `Task{ID, Title, Status, CreatedAt}`, `TaskStatus` (todo/doing/done)

**Use Case interface**: `TaskUseCase` — Create, GetByID, List, UpdateStatus, Delete

**Repository interface**: `TaskRepository` — Save, FindByID, FindAll, Update, Delete

**In-Memory Repository**: реализация `TaskRepository` на map

**Use Case Implementation**: бизнес-логика + валидация

Зависимости идут только внутрь: handler → usecase → repository.
