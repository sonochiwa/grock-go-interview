# AST Parsing

## Обзор

Пакеты `go/ast`, `go/parser`, `go/token` позволяют анализировать Go код программно. Основа для линтеров, генераторов, рефакторинга.

```go
import (
    "go/ast"
    "go/parser"
    "go/token"
)

func main() {
    fset := token.NewFileSet()
    node, err := parser.ParseFile(fset, "example.go", nil, parser.ParseComments)
    if err != nil {
        log.Fatal(err)
    }

    // Обход AST
    ast.Inspect(node, func(n ast.Node) bool {
        if fn, ok := n.(*ast.FuncDecl); ok {
            fmt.Printf("Function: %s\n", fn.Name.Name)
        }
        return true
    })
}
```

### Применения

- Кастомные линтеры
- Генерация кода на основе интерфейсов
- Автоматический рефакторинг
- Документация
