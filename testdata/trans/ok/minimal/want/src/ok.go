package main

import (
	"os"
)

func f() error {
	if err := os.Setenv("hello", "world"); err != nil {
		return err
	}
	return nil
}

func main() {
	f()
}
