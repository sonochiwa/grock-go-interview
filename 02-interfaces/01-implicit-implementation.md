# Неявная реализация интерфейсов

## Обзор

В Go нет ключевого слова `implements`. Тип реализует интерфейс, если имеет все его методы. Это называют "duck typing" — если ходит как утка и крякает как утка, значит это утка.

## Концепции

### Базовый пример

```go
type Writer interface {
    Write(p []byte) (n int, err error)
}

// os.File реализует Writer (имеет метод Write)
// bytes.Buffer реализует Writer
// *net.TCPConn реализует Writer
// Ни один из них не объявляет "implements Writer"

// Любой тип с методом Write([]byte)(int, error) — Writer:
type MyWriter struct{}

func (w MyWriter) Write(p []byte) (int, error) {
    fmt.Println(string(p))
    return len(p), nil
}

var w Writer = MyWriter{} // OK — неявно реализует Writer
```

### Compile-time проверка реализации

```go
// Трюк: убедиться на этапе компиляции, что тип реализует интерфейс
var _ Writer = (*MyWriter)(nil) // не аллоцирует, проверяется компилятором

// Если MyWriter не реализует Writer — ошибка компиляции
```

### Маленькие интерфейсы

Go продвигает принцип **маленьких интерфейсов** (1-3 метода):

```go
// Стандартная библиотека — примеры:
type Reader interface { Read(p []byte) (n int, err error) }
type Writer interface { Write(p []byte) (n int, err error) }
type Closer interface { Close() error }
type Stringer interface { String() string }

// Композиция интерфейсов
type ReadWriter interface {
    Reader
    Writer
}

type ReadWriteCloser interface {
    Reader
    Writer
    Closer
}
```

**Rob Pike:** *"The bigger the interface, the weaker the abstraction."*

### Принимай интерфейсы, возвращай структуры

```go
// ХОРОШО: функция принимает интерфейс
func ProcessData(r io.Reader) error {
    data, err := io.ReadAll(r)
    // ...
}
// Можно передать файл, HTTP body, буфер, сокет...

// ХОРОШО: функция возвращает конкретный тип
func NewUserService(db *sql.DB) *UserService {
    return &UserService{db: db}
}
// Вызывающий код знает точный тип, может использовать все методы

// ПЛОХО: возвращать интерфейс (в большинстве случаев)
func NewUserService(db *sql.DB) UserServiceInterface { ... }
// Скрывает реализацию, затрудняет тестирование и навигацию
```

### Value receiver vs Pointer receiver для интерфейсов

```go
type Sayer interface {
    Say() string
}

type Dog struct{ Name string }
func (d Dog) Say() string { return "Woof! I'm " + d.Name }

type Cat struct{ Name string }
func (c *Cat) Say() string { return "Meow! I'm " + c.Name }

var s Sayer

s = Dog{Name: "Rex"}   // OK — value receiver: и Dog, и *Dog реализуют Sayer
s = &Dog{Name: "Rex"}  // OK

s = &Cat{Name: "Tom"}  // OK — pointer receiver: только *Cat реализует Sayer
// s = Cat{Name: "Tom"} // ОШИБКА: Cat не реализует Sayer (только *Cat)
```

**Почему?** Value receiver метод может быть вызван и на значении, и на указателе (Go автоматически берёт адрес). Но pointer receiver метод не может быть вызван на значении, хранящемся в интерфейсе — потому что нельзя взять адрес значения внутри интерфейса.

## Частые вопросы на собеседованиях

**Q: Почему в Go неявная реализация интерфейсов?**
A: Снижает связность (coupling). Тип не зависит от пакета с интерфейсом. Можно определить интерфейс после написания реализации. Разные пакеты могут определять интерфейсы для одних и тех же типов.

**Q: Где определять интерфейс — в пакете реализации или использования?**
A: В пакете **использования** (consumer). Это позволяет определять только нужные методы и снижает зависимости.

**Q: Почему Cat{} не реализует интерфейс, если метод определён на *Cat?**
A: Потому что значение внутри интерфейса не адресуемо. Go не может автоматически взять адрес значения, хранящегося в интерфейсе.

**Q: Как проверить реализацию интерфейса в compile-time?**
A: `var _ Interface = (*Type)(nil)` — не аллоцирует, ошибка если тип не соответствует.

## Подводные камни

1. **Забыл pointer receiver** — тип не реализует интерфейс, ошибка только при присваивании.

2. **Слишком большие интерфейсы** — если интерфейс имеет 10 методов, его сложно реализовать и мокать.

3. **Определение интерфейса заранее** — в Go идиоматично определять интерфейсы по мере необходимости, а не "на вырост".
