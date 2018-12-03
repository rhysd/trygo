package main

import (
	"os"
)

func f() error {
	try(os.Setenv("hello", "world"))
	return nil
}

func main() {
	f()
}
