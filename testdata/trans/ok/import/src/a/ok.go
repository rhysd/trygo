package a

import (
	"fmt"
)

func Foo() (int, error) {
	n := try(fmt.Println("hello"))
	return n + 10, nil
}
