package foo

import (
	"fmt"
)

func f() (int, error) {
	n := 42
	n += try(fmt.Println("hello"))
	return n, nil
}
