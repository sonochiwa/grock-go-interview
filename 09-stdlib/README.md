# 09. Стандартная библиотека Go

Ключевые пакеты stdlib, которые часто спрашивают на собеседованиях.

## Содержание

| #  | Файл | Темы |
|----|------|------|
| 01 | [net/http](01-net-http.md) | Server, Client, Handler, Middleware, http.ServeMux (1.22+), timeouts, context |
| 02 | [encoding/json](02-encoding-json.md) | Marshal/Unmarshal, struct tags, custom (Un)Marshaler, streaming (Decoder/Encoder), omitempty, `json.Number` |
| 03 | [io](03-io.md) | Reader, Writer, ReadCloser, Pipe, MultiReader, TeeReader, io.Copy, io.LimitReader |
| 04 | [bytes и strings](04-bytes-strings.md) | Builder, Buffer, strings.Builder, strings vs []byte, immutability |
| 05 | [sort и slices](05-sort-slices.md) | sort.Interface, sort.Slice, slices.SortFunc (1.21+), binary search |
| 06 | [time](06-time.md) | Time, Duration, Ticker, Timer, After, time.Parse, time zones, monotonic clock |
| 07 | [os и filepath](07-os-filepath.md) | файловая система, переменные окружения, сигналы, os.Exit vs log.Fatal |
| 08 | [fmt](08-fmt.md) | verbs (%v, %+v, %#v, %T), Stringer, GoStringer, Formatter, Errorf |
| 09 | [maps и cmp](09-maps-cmp.md) | maps.Clone, maps.Keys, cmp.Or, cmp.Compare (1.21+) |
| 10 | [HTTP/REST паттерны](10-http-patterns.md) | REST, middleware, routing (ServeMux 1.22+), request/response, graceful shutdown, тестирование |

---

## Задачи

Практические задачи по этой теме: [exercises/](exercises/)
