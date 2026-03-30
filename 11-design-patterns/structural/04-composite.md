# Composite

## В Go

Composite — древовидная структура, где узлы и листья реализуют один интерфейс.

```go
type Component interface {
    Execute() string
}

// Лист
type File struct{ name string }
func (f *File) Execute() string { return f.name }

// Узел (содержит дочерние компоненты)
type Folder struct {
    name     string
    children []Component
}

func (f *Folder) Execute() string {
    var results []string
    for _, child := range f.children {
        results = append(results, child.Execute())
    }
    return f.name + ": [" + strings.Join(results, ", ") + "]"
}

func (f *Folder) Add(c Component) { f.children = append(f.children, c) }

// Использование
root := &Folder{name: "root"}
root.Add(&File{name: "main.go"})
sub := &Folder{name: "pkg"}
sub.Add(&File{name: "utils.go"})
root.Add(sub)
root.Execute() // "root: [main.go, pkg: [utils.go]]"
```
