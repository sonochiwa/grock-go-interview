package main

import (
	"errors"
	"fmt"
	"math/rand"
)

var SentinelError = errors.New("sentinel error")

func somethingWrong() bool {
	return rand.Intn(2) == 1
}

func foo() error {
	if somethingWrong() {
		return SentinelError
	}

	return fmt.Errorf("just new error")
}

func main() {
	err := foo()
	if err != nil {
		if errors.Is(err, SentinelError) {
			fmt.Printf("handled error: %v\n", err.Error())
			return
		}

		fmt.Printf("unhandled error: %v\n", err.Error())
		return
	}
}
