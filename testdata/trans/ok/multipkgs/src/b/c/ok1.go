package c

import (
	"fmt"
)

func Foo() (int, error) {
	x := try(fmt.Println("hello"))
	x = try(fmt.Println("bye"))
	return x, nil
}
