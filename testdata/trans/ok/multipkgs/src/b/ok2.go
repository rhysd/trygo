package b

import (
	"fmt"
)

func Bar() (string, error) {
	try(Foo())
	n := try(Foo())
	n = try(Foo())
	return fmt.Sprint(n), nil
}
