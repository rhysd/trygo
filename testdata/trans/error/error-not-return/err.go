package foo

import (
	"fmt"
)

func f() int {
	n := try(fmt.Println("hello"))
	return n
}
