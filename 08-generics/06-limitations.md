# Ограничения дженериков

## Обзор

Go дженерики намеренно ограничены. Знание ограничений помогает выбрать правильный подход.

## Ограничения

### 1. Нет type parameters на методах

```go
type Container struct{}

// ОШИБКА: методы не могут иметь свои type parameters
func (c Container) Map[T, R any](f func(T) R) []R { ... }

// Workaround: функция вместо метода
func Map[T, R any](c Container, f func(T) R) []R { ... }

// Или type parameter на типе
type Container[T any] struct{ items []T }
func (c Container[T]) Filter(f func(T) bool) Container[T] { ... }
```

### 2. Нет специализации

```go
// Нельзя написать разную реализацию для разных типов
func Process[T any](v T) {
    // Нет способа сделать "если T == string, то ..."
}

// Workaround: type switch через any
func Process[T any](v T) {
    switch val := any(v).(type) {
    case string: // ...
    case int:    // ...
    }
}
```

### 3. Нет variadic type parameters

```go
// Нельзя: func Zip[T1, T2, T3, ... any](...)
// Нужно создавать отдельные функции Zip2, Zip3, ...
```

### 4. Нет generic constraints на число аргументов

### 5. Нет covariance/contravariance

```go
type Animal struct{}
type Dog struct{ Animal }

// []Dog НЕ является []Animal
// Stack[Dog] НЕ является Stack[Animal]
```

## Частые вопросы на собеседованиях

**Q: Почему методы не могут иметь type parameters?**
A: Усложнило бы реализацию интерфейсов и dispatch. Go команда решила не добавлять это в первой итерации.

**Q: Когда НЕ использовать дженерики?**
A: Когда обычные интерфейсы работают, когда код становится сложнее для чтения, когда нужна специализация.
