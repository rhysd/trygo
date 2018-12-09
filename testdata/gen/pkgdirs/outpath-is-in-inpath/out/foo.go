package main

import (
	"fmt"
	"os"
)

func f() (int, error) {
	var n int
	cwd, _err0 := os.Getwd()
	if _err0 != nil {
		return 0, _err0
	}
	var f, _err1 = os.Create(cwd + "/foo")
	if _err1 != nil {
		return 0, _err1
	}
	defer f.Close()
	var _err2 error
	n, _err2 = f.Write([]byte("hello\n"))
	if _err2 != nil {
		return 0, _err2
	}
	fmt.Println("Wrote:", n)
	return n, nil
}

func main() {
	f()
}
