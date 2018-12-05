package foo

import (
	"fmt"
)

func Foo(n int) error {
	if _, err := fmt.Println(n); err != nil {
		return err
	}
	return nil
}

func Bar() (int, error) {
	var err = Foo(10)
	if err != nil {
		return 0, err
	}
	err = Foo(111)
	if err != nil {
		return 0, err
	}
	err, err = Foo(1), Foo(2)
	return 42, nil
}
