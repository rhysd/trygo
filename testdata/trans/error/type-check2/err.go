package foo

import (
	"fmt"
)

func f() (int, error) {
	n := try(fmt.Println("hello"))
	return fmt.Sprint(n), nil
}

func g() {
	// unused
	x := 42
}
