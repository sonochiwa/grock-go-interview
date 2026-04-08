# Facade

## В Go

Facade предоставляет простой интерфейс к сложной подсистеме.

```go
// Сложная подсистема
type UserRepo struct { db *sql.DB }
type EmailService struct { smtp *SMTPClient }
type AuditLog struct { logger *slog.Logger }

// Facade: простой API для регистрации
type RegistrationService struct {
    users  *UserRepo
    emails *EmailService
    audit  *AuditLog
}

func (s *RegistrationService) Register(ctx context.Context, name, email string) error {
    user, err := s.users.Create(ctx, name, email)
    if err != nil {
        return fmt.Errorf("create user: %w", err)
    }

    if err := s.emails.SendWelcome(ctx, user); err != nil {
        s.audit.Log("welcome email failed", "user", user.ID)
        // не возвращаем ошибку — email не критичен
    }

    s.audit.Log("user registered", "user", user.ID)
    return nil
}

// Вызывающий код не знает о UserRepo, EmailService, AuditLog
err := registrationService.Register(ctx, "Alice", "alice@example.com")
```

В Go facade — просто структура, агрегирующая зависимости с простым публичным API.
