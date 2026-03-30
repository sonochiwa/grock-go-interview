# Authorization

## RBAC (Role-Based Access Control)

```go
type Role string

const (
    RoleAdmin     Role = "admin"
    RoleModerator Role = "moderator"
    RoleUser      Role = "user"
)

type Permission string

const (
    PermCreateUser  Permission = "user:create"
    PermReadUser    Permission = "user:read"
    PermUpdateUser  Permission = "user:update"
    PermDeleteUser  Permission = "user:delete"
    PermManageRoles Permission = "role:manage"
)

var rolePermissions = map[Role][]Permission{
    RoleAdmin:     {PermCreateUser, PermReadUser, PermUpdateUser, PermDeleteUser, PermManageRoles},
    RoleModerator: {PermReadUser, PermUpdateUser},
    RoleUser:      {PermReadUser},
}

func HasPermission(role Role, perm Permission) bool {
    perms, ok := rolePermissions[role]
    if !ok {
        return false
    }
    for _, p := range perms {
        if p == perm {
            return true
        }
    }
    return false
}

// Middleware
func requirePermission(perm Permission) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims := claimsFromContext(r.Context())
            if claims == nil {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }
            if !HasPermission(Role(claims.Role), perm) {
                http.Error(w, "forbidden", http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// Router
r.With(requirePermission(PermDeleteUser)).Delete("/users/{id}", handler.DeleteUser)
r.With(requirePermission(PermReadUser)).Get("/users/{id}", handler.GetUser)
```

## ABAC (Attribute-Based Access Control)

```go
// Более гибкий чем RBAC — решение на основе атрибутов
type Policy struct {
    Resource string
    Action   string
    Check    func(ctx context.Context, subject Subject, resource any) bool
}

type Subject struct {
    UserID string
    Role   Role
    TeamID string
}

var policies = []Policy{
    {
        Resource: "order",
        Action:   "cancel",
        Check: func(ctx context.Context, subj Subject, res any) bool {
            order := res.(*Order)
            // Admin может отменить любой заказ
            if subj.Role == RoleAdmin {
                return true
            }
            // Пользователь может отменить только свой
            return order.UserID == subj.UserID
        },
    },
    {
        Resource: "document",
        Action:   "edit",
        Check: func(ctx context.Context, subj Subject, res any) bool {
            doc := res.(*Document)
            // Владелец или участник команды
            return doc.OwnerID == subj.UserID || doc.TeamID == subj.TeamID
        },
    },
}

func Authorize(ctx context.Context, subj Subject, resource string, action string, obj any) bool {
    for _, p := range policies {
        if p.Resource == resource && p.Action == action {
            return p.Check(ctx, subj, obj)
        }
    }
    return false // deny by default
}

// Использование
if !Authorize(ctx, subject, "order", "cancel", order) {
    return status.Error(codes.PermissionDenied, "cannot cancel this order")
}
```

## Resource-Based Authorization в handler

```go
func (h *Handler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
    claims := claimsFromContext(r.Context())
    orderID := chi.URLParam(r, "id")

    order, err := h.repo.Get(r.Context(), orderID)
    if err != nil {
        respondError(w, http.StatusNotFound, "order not found")
        return
    }

    // Проверка владельца
    if order.UserID != claims.UserID && claims.Role != "admin" {
        respondError(w, http.StatusForbidden, "not your order")
        return
    }

    // Обновление...
}
```

## API Key Authentication

```go
// Для service-to-service или third-party integrations
func apiKeyMiddleware(validKeys map[string]string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            key := r.Header.Get("X-API-Key")
            if key == "" {
                key = r.URL.Query().Get("api_key")
            }

            // Constant-time comparison
            clientName, valid := validateAPIKey(validKeys, key)
            if !valid {
                http.Error(w, "invalid API key", http.StatusUnauthorized)
                return
            }

            ctx := context.WithValue(r.Context(), clientNameKey, clientName)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func validateAPIKey(validKeys map[string]string, providedKey string) (string, bool) {
    for name, key := range validKeys {
        if subtle.ConstantTimeCompare([]byte(key), []byte(providedKey)) == 1 {
            return name, true
        }
    }
    return "", false
}
```

## Best Practices

```
1. Deny by default — если не разрешено явно, запрещено
2. Principle of least privilege — минимум прав
3. Check authorization at every layer:
   - API Gateway: auth token valid?
   - Handler: has permission?
   - Service: owns resource?
4. Audit log — кто что сделал когда
5. Не доверяй client-side:
   - Роль из JWT → verify на сервере
   - Права из localStorage → бесполезно
```
