package main

import (
	"fmt"
	"os"
)

func f() (int, error) {
	var n int
	cwd := try(os.Getwd())
	var f = try(os.Create(cwd + "/foo"))
	defer f.Close()
	n = try(f.Write([]byte("hello\n")))
	fmt.Println("Wrote:", n)
	return n, nil
}

func main() {
	f()
}
