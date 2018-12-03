package trygo_test

import (
	"bytes"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rhysd/trygo"
)

func replaceLast(s, old, new string) string {
	idx := strings.LastIndex(s, old)
	if idx == -1 {
		return s
	}
	return s[:idx] + new + s[idx+len(old):]
}

func collectPackagesUnder(dirpath string, t *testing.T) []*trygo.Package {
	fset := token.NewFileSet()
	saw := map[string]*trygo.Package{}
	if err := filepath.Walk(dirpath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(p, ".go") {
			return nil
		}
		dir := filepath.Dir(p)
		if _, ok := saw[dir]; ok {
			return nil
		}
		pkgs, err := parser.ParseDir(fset, dir, nil, 0)
		if err != nil {
			t.Fatal(dirpath, err)
		}
		for p, pkg := range pkgs {
			dest := replaceLast(dir, filepath.FromSlash("/src"), filepath.FromSlash("/want/src"))
			saw[p] = trygo.NewPackage(pkg, dir, dest, fset)
		}
		return nil
	}); err != nil {
		t.Fatal(dirpath, err)
	}
	if len(saw) == 0 {
		t.Fatal("0 package found under directory " + dirpath)
	}
	ret := make([]*trygo.Package, 0, len(saw))
	for _, pkg := range saw {
		ret = append(ret, pkg)
	}
	return ret
}

func TestTranslationOK(t *testing.T) {
	base := filepath.Join(cwd, "testdata", "trans", "ok")
	entries, err := ioutil.ReadDir(base)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dir := filepath.Join(base, entry.Name())
		t.Run(entry.Name(), func(t *testing.T) {
			pkgs := collectPackagesUnder(filepath.Join(dir, "src"), t)
			if err := trygo.Translate(pkgs); err != nil {
				t.Fatal(err)
			}

			for _, pkg := range pkgs {
				shouldModified := ""

				fs, err := ioutil.ReadDir(pkg.Path)
				if err != nil {
					t.Fatal("Want dir does not exist", err, pkg.Path, pkg.Node.Name)
				}

				// Check all wanted files were translated and their contents are as expected
				for _, f := range fs {
					name := f.Name()
					if f.IsDir() || !strings.HasSuffix(name, ".go") {
						continue
					}

					path := filepath.Join(pkg.Path, name)
					if !strings.HasPrefix(name, "skip") {
						shouldModified = path
					}

					var buf bytes.Buffer
					if err := pkg.WriteFileTo(&buf, path); err != nil {
						t.Fatal(err, pkg.Node.Files)
					}
					have := buf.String()

					b, err := ioutil.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					want := string(b)

					if want != have {
						t.Fatalf("Translated source is unexpected.\nWanted:\n%s\n\nHave:\n%s\n", want, have)
					}
				}

				if shouldModified == "" {
					if pkg.Modified() {
						t.Fatal("No file was modified but pkg.Modified() returned true", pkg)
					}
				} else {
					if !pkg.Modified() {
						t.Fatal(shouldModified, "was modified but pkg.Modified() returned false", pkg)
					}
				}
			}
		})
	}
}
