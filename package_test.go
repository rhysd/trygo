package trygo_test

import (
	"bytes"
	"github.com/rhysd/trygo"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
)

func testPackageParseDir(t *testing.T, dir string) (*token.FileSet, *ast.Package) {
	fs := token.NewFileSet()
	pkgs, err := parser.ParseDir(fs, dir, nil, 0)
	if err != nil {
		t.Fatalf("Parse error at %q: %s", dir, err)
	}
	for _, pkg := range pkgs {
		return fs, pkg
	}
	t.Fatal("No pacakge at", dir)
	return nil, nil
}

func TestPackageWriteToBufferOK(t *testing.T) {
	dir := filepath.Join(cwd, "testdata", "package", "normal")
	fs, pkg := testPackageParseDir(t, dir)
	dest := filepath.Join(dir, "dest")
	p := trygo.NewPackage(pkg, dir, dest, fs)
	if err := p.Verify(); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	file := filepath.Join(dir, "foo.go")
	if err := p.WriteFileTo(&buf, file); err != nil {
		t.Fatal(err)
	}
	have := buf.String()

	b, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	want := string(b)

	if have != want {
		t.Fatalf("Written Go file is different from original:\nWant:\n%s\n\nHave:\n%s", want, have)
	}
}

func TestPackageWriteToBufferError(t *testing.T) {
	dir := filepath.Join(cwd, "testdata", "package", "normal")
	fs, pkg := testPackageParseDir(t, dir)
	dest := filepath.Join(dir, "dest")
	p := trygo.NewPackage(pkg, dir, dest, fs)
	err := p.WriteFileTo(ioutil.Discard, "/path/to/unknown/file")
	if err == nil || !strings.Contains(err.Error(), "No file translated") {
		t.Fatal("Unexpected error:", err)
	}
}

func TestPackageVerifyFailure(t *testing.T) {
	dir := filepath.Join(cwd, "testdata", "package", "broken")
	fs, pkg := testPackageParseDir(t, dir)
	dest := filepath.Join(dir, "dest")
	p := trygo.NewPackage(pkg, dir, dest, fs)
	err := p.Verify()
	if err == nil || !strings.Contains(err.Error(), "Type error(s) at verification after translation") {
		t.Fatal("Error unexpected:", err)
	}
}
