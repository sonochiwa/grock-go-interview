# Стандартные интерфейсы Go

## Обзор

Знание ключевых интерфейсов стандартной библиотеки — показатель опыта. Эти интерфейсы маленькие, но на них построена вся экосистема.

## io.Reader и io.Writer

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}
```

Реализуют: *os.File, *bytes.Buffer, *strings.Reader, *net.TCPConn, *http.Response.Body, *gzip.Reader, *bufio.Reader...

```go
// Сила абстракции: одна функция работает с любым источником
func countLines(r io.Reader) (int, error) {
    scanner := bufio.NewScanner(r)
    count := 0
    for scanner.Scan() {
        count++
    }
    return count, scanner.Err()
}

// Можно передать файл, HTTP body, строку, сжатый поток...
countLines(os.Stdin)
countLines(strings.NewReader("line1\nline2"))
countLines(resp.Body)
```

### Композиция io интерфейсов

```go
type ReadWriter interface { Reader; Writer }
type ReadCloser interface { Reader; Closer }
type WriteCloser interface { Writer; Closer }
type ReadWriteCloser interface { Reader; Writer; Closer }

// io.Closer
type Closer interface { Close() error }

// io.Seeker
type Seeker interface { Seek(offset int64, whence int) (int64, error) }
```

### Утилиты io пакета

```go
io.Copy(dst Writer, src Reader)          // копирует всё
io.ReadAll(r Reader) ([]byte, error)     // читает всё в память
io.LimitReader(r Reader, n int64) Reader // ограничивает чтение
io.TeeReader(r Reader, w Writer) Reader  // читает и пишет одновременно
io.MultiReader(readers ...Reader) Reader // конкатенация
io.MultiWriter(writers ...Writer) Writer // дублирование записи
```

## fmt.Stringer

```go
type Stringer interface {
    String() string
}

// fmt.Println, fmt.Sprintf и т.д. вызывают String() автоматически
type User struct {
    Name string
    Age  int
}

func (u User) String() string {
    return fmt.Sprintf("%s (%d)", u.Name, u.Age)
}

fmt.Println(User{"Alice", 25}) // "Alice (25)"
```

## error

```go
type error interface {
    Error() string
}

// Любой тип с методом Error() string — ошибка
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("%s: %s", e.Field, e.Message)
}
```

## sort.Interface

```go
type Interface interface {
    Len() int
    Less(i, j int) bool
    Swap(i, j int)
}

// С Go 1.21 проще через slices.SortFunc:
slices.SortFunc(users, func(a, b User) int {
    return cmp.Compare(a.Age, b.Age)
})
```

## encoding интерфейсы

```go
// json.Marshaler / json.Unmarshaler
type Marshaler interface { MarshalJSON() ([]byte, error) }
type Unmarshaler interface { UnmarshalJSON([]byte) error }

// encoding.TextMarshaler / encoding.TextUnmarshaler
type TextMarshaler interface { MarshalText() ([]byte, error) }

// Пример: кастомная сериализация
type Status int
const (
    Active Status = iota
    Inactive
)

func (s Status) MarshalJSON() ([]byte, error) {
    switch s {
    case Active:
        return []byte(`"active"`), nil
    case Inactive:
        return []byte(`"inactive"`), nil
    default:
        return nil, fmt.Errorf("unknown status: %d", s)
    }
}
```

## http.Handler

```go
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}

// http.HandlerFunc — адаптер для функций
type HandlerFunc func(ResponseWriter, *Request)
func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) { f(w, r) }

// Это позволяет использовать обычные функции как Handler:
http.Handle("/", http.HandlerFunc(myHandler))
```

## context.Context

```go
type Context interface {
    Deadline() (deadline time.Time, ok bool)
    Done() <-chan struct{}
    Err() error
    Value(key any) any
}
```

## Частые вопросы на собеседованиях

**Q: Зачем нужен io.Reader, а не просто []byte?**
A: Потоковая обработка — не нужно загружать весь файл/поток в память. Абстракция — один код работает с файлами, сетью, буферами.

**Q: Как http.HandlerFunc реализует Handler, если это функция?**
A: Это type definition для `func(ResponseWriter, *Request)` с методом ServeHTTP, который вызывает саму функцию. Паттерн "adapter".

**Q: Почему error — это интерфейс, а не конкретный тип?**
A: Позволяет создавать ошибки с дополнительным контекстом (поля, методы), сохраняя единый способ обработки через `if err != nil`.
