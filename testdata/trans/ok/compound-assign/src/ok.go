package foo

import (
	"fmt"
)

func f() error {
	n := try(fmt.Println("hello"))
	n += try(fmt.Println("hello"))
	n -= try(fmt.Println("hello"))
	n *= try(fmt.Println("hello"))
	n /= try(fmt.Println("hello"))
	n %= try(fmt.Println("hello"))
	n &= try(fmt.Println("hello"))
	return nil
}

func g() (int, error) {
	n := try(fmt.Println("hello"))
	n += try(fmt.Println("hello"))
	n -= try(fmt.Println("hello"))
	n *= try(fmt.Println("hello"))
	n /= try(fmt.Println("hello"))
	n %= try(fmt.Println("hello"))
	n &= try(fmt.Println("hello"))
	return n, nil
}
