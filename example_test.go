package trygo_test

import (
	"fmt"
	"github.com/rhysd/trygo"
	"io/ioutil"
	"path/filepath"
)

func ExampleGen() {
	pkgDir := filepath.Join("testdata", "example")
	outDir := filepath.Join(pkgDir, "out")

	// Create a code generator for TryGo to Go translation
	gen, err := trygo.NewGen(outDir)
	if err != nil {
		panic(err)
	}

	// Generate() outputs translated file paths by default. If you don't want them, please set
	// ioutil.Discard as writer.
	gen.Writer = ioutil.Discard

	// Translate TryGo package into Go and generate it at outDir with output verification.
	// It generates testdata/example/out.
	if err := gen.Generate([]string{pkgDir}, true); err != nil {
		panic(err)
	}

	fmt.Println("OK")
	// Output: OK
}
