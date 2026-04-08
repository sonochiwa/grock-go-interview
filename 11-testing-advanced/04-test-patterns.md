# Test Patterns

## Golden Files

```go
// Golden file = ожидаемый output сохранён в файле
// При -update флаге — перезаписывается

var update = flag.Bool("update", false, "update golden files")

func TestRenderTemplate(t *testing.T) {
    result := renderTemplate(inputData)

    golden := filepath.Join("testdata", t.Name()+".golden")

    if *update {
        os.WriteFile(golden, []byte(result), 0644)
        return
    }

    expected, err := os.ReadFile(golden)
    require.NoError(t, err)
    assert.Equal(t, string(expected), result)
}

// Обновление golden files:
// go test -run TestRenderTemplate -update
```

## Test Helpers

```go
// t.Helper() — помечает функцию как helper
// При ошибке показывает строку вызова, а не строку внутри helper
func assertUserEqual(t *testing.T, expected, actual *User) {
    t.Helper()
    assert.Equal(t, expected.Name, actual.Name)
    assert.Equal(t, expected.Email, actual.Email)
}

// Cleanup
func createTempDir(t *testing.T) string {
    t.Helper()
    dir := t.TempDir() // автоматически удаляется после теста
    return dir
}

// Test fixture
func loadFixture(t *testing.T, name string) []byte {
    t.Helper()
    data, err := os.ReadFile(filepath.Join("testdata", name))
    require.NoError(t, err)
    return data
}
```

## Subtests и Parallel

```go
func TestUserService(t *testing.T) {
    svc := setupService(t)

    // Subtests для группировки
    t.Run("Create", func(t *testing.T) {
        t.Parallel() // параллельно с другими Parallel subtests

        user, err := svc.Create(ctx, "Alice")
        require.NoError(t, err)
        assert.NotEmpty(t, user.ID)
    })

    t.Run("Get", func(t *testing.T) {
        t.Parallel()

        // ...
    })

    // Table-driven + parallel
    tests := []struct {
        name  string
        input string
        want  error
    }{
        {"empty name", "", ErrValidation},
        {"valid", "Alice", nil},
        {"too long", strings.Repeat("a", 256), ErrValidation},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            _, err := svc.Create(ctx, tt.input)
            assert.ErrorIs(t, err, tt.want)
        })
    }
}
```

## Custom Test Main

```go
// Общий setup для всего пакета
func TestMain(m *testing.M) {
    // Setup
    log.Println("setting up tests...")

    // Можно парсить флаги
    flag.Parse()

    code := m.Run()

    // Teardown
    log.Println("cleaning up...")

    os.Exit(code)
}
```

## testdata/ директория

```
Правила:
  - go tool игнорирует testdata/ (не компилирует)
  - Используй для: fixtures, golden files, test configs, sample files
  - Путь: относительно файла теста

mypackage/
├── handler.go
├── handler_test.go
└── testdata/
    ├── valid_request.json
    ├── invalid_request.json
    ├── TestRenderTemplate/
    │   └── basic.golden
    └── schema.sql
```

## Custom Assertions

```go
// Для domain-specific проверок
func assertValidOrder(t *testing.T, order *Order) {
    t.Helper()
    assert.NotEmpty(t, order.ID, "order ID should not be empty")
    assert.Positive(t, order.Total, "order total should be positive")
    assert.NotEmpty(t, order.Items, "order should have items")
    assert.False(t, order.CreatedAt.IsZero(), "created_at should be set")

    var itemsTotal int
    for _, item := range order.Items {
        itemsTotal += item.Price * item.Quantity
    }
    assert.Equal(t, itemsTotal, order.Total, "total should match sum of items")
}
```

## Skip Patterns

```go
func TestRequiresDocker(t *testing.T) {
    if os.Getenv("DOCKER_HOST") == "" {
        if _, err := exec.LookPath("docker"); err != nil {
            t.Skip("docker not available")
        }
    }
    // ...
}

func TestRequiresEnv(t *testing.T) {
    apiKey := os.Getenv("API_KEY")
    if apiKey == "" {
        t.Skip("API_KEY not set")
    }
    // ...
}

func TestSlow(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping slow test")
    }
    // ... тест на 30 секунд
}
```
