package foo

import (
	"fmt"
)

func f() (int, error) {
	n := try(fmt.Println("hello"))
	if x := n + 10; x != 11 {
		return x, nil
	}

	n = try(fmt.Println("bye"))
	return n, nil
}
