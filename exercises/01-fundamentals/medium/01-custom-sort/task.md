# Custom Sort

Реализуй функцию `SortBy[T any](s []T, less func(a, b T) bool) []T`, которая сортирует копию слайса. Также напиши `SortByField` для сортировки слайса структур по имени поля (используя reflect).

## Требования

- `SortBy` возвращает отсортированную копию, исходный слайс не мутируется
- `SortByField[T any](s []T, fieldName string, ascending bool) ([]T, error)` использует reflect
- `SortByField` возвращает ошибку если поле не найдено или не comparable
- Поддержка сортировки по полям типов: int, float64, string

## Пример

```go
nums := []int{3, 1, 4, 1, 5}
sorted := SortBy(nums, func(a, b int) bool { return a < b })
// sorted = [1, 1, 3, 4, 5]

type Person struct {
    Name string
    Age  int
}
people := []Person{{Name: "Bob", Age: 30}, {Name: "Alice", Age: 25}}
sorted, _ := SortByField(people, "Age", true)
// sorted = [{Alice 25}, {Bob 30}]
```
