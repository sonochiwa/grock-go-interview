package error_wrapping

import (
	"errors"
	"fmt"
	"net/http"
)

var ErrNotFound = errors.New("not found")

type User struct {
	ID   int
	Name string
}

var users = map[int]User{
	1: {ID: 1, Name: "Alice"},
	2: {ID: 2, Name: "Bob"},
}

func GetUser(id int) (User, error) {
	u, ok := users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func GetUserService(id int) (User, error) {
	u, err := GetUser(id)
	if err != nil {
		return User{}, fmt.Errorf("get user id=%d: %w", id, err)
	}
	return u, nil
}

func HandleGetUser(id int) (User, int, error) {
	u, err := GetUserService(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return User{}, http.StatusNotFound, err
		}
		return User{}, http.StatusInternalServerError, err
	}
	return u, http.StatusOK, nil
}
