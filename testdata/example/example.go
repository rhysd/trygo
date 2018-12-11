package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func CreateFileInSubdir(subdir, filename string, content []byte) error {
	cwd := try(os.Getwd())

	try(os.Mkdir(filepath.Join(cwd, subdir), 0755))

	p := filepath.Join(cwd, subdir, filename)
	f := try(os.Create(p))
	defer f.Close()

	try(f.Write(content))

	fmt.Println("Created:", p)
	return nil
}

func main() {
	if err := CreateFileInSubdir("foo", "test.txt", []byte("hello\n")); err != nil {
		panic(err)
	}
}
