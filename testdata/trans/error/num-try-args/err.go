package foo

import (
	"fmt"
)

func f() error {
	try(fmt.Println("hello"), "oops", true)
}
