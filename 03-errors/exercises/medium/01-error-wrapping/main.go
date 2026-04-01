package error_wrapping

import "errors"

var ErrNotFound = errors.New("not found")

type User struct {
	ID   int
	Name string
}

// Хранилище пользователей (in-memory)
var users = map[int]User{
	1: {ID: 1, Name: "Alice"},
	2: {ID: 2, Name: "Bob"},
}

// TODO: возвращает ErrNotFound если пользователь не найден
func GetUser(id int) (User, error) {
	return User{}, nil
}

// TODO: вызывает GetUser, оборачивает ошибку с контекстом через fmt.Errorf + %w
func GetUserService(id int) (User, error) {
	return User{}, nil
}

// TODO: вызывает GetUserService, возвращает (user, httpStatus, error)
// ErrNotFound → 404, другая ошибка → 500, успех → 200
func HandleGetUser(id int) (User, int, error) {
	return User{}, 0, nil
}
