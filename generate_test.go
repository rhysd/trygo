package trygo_test

import (
	"github.com/rhysd/go-fakeio"
	"github.com/rhysd/go-tmpenv"
	"github.com/rhysd/trygo"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateOK(t *testing.T) {
	base := filepath.Join("testdata", "gen", "ok")
	es, err := ioutil.ReadDir(base)
	if err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(base, "HAVE")

	for _, e := range es {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "WANT" || name == "HAVE" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			fake := fakeio.Stdout()
			defer fake.Restore()

			os.RemoveAll(filepath.Join(base, "HAVE", name))
			dir := filepath.Join(base, name)
			gen, err := trygo.NewGen(outDir)
			if err != nil {
				t.Fatal(err)
			}
			if err := gen.Generate([]string{dir}); err != nil {
				t.Fatal(err)
			}
			if s, err := os.Stat(outDir); err != nil || !s.IsDir() {
				t.Fatal(err, s)
			}

			paths := []string{}
			if err := filepath.Walk(filepath.Join(base, "WANT", name), func(wantPath string, info os.FileInfo, err error) error {
				if err != nil {
					t.Fatal(wantPath, err)
				}
				if info.IsDir() {
					return nil
				}
				if !strings.HasSuffix(wantPath, ".go") {
					t.Fatal("Not Go source", wantPath)
				}
				havePath := strings.Replace(wantPath, filepath.FromSlash("/WANT/"), filepath.FromSlash("/HAVE/"), 1)
				b, err := ioutil.ReadFile(wantPath)
				if err != nil {
					t.Fatal(err)
				}
				want := string(b)
				b, err = ioutil.ReadFile(havePath)
				if err != nil {
					t.Fatal(err)
				}
				have := string(b)
				if want != have {
					t.Fatalf("Translation result does not match at %s\nwanted:\n%s\nbut have:\n%s\n", havePath, want, have)
				}
				paths = append(paths, havePath)
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			stdout, err := fake.String()
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range paths {
				if !strings.Contains(stdout, path) {
					t.Error(path, "is not contained in stdout:", stdout)
				}
			}
		})
	}
}

func TestGenerateGoGenerateOK(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	base := filepath.FromSlash("testdata/gen/ok/simple/")
	if err := os.Chdir(base); err != nil {
		t.Fatal(err)
	}

	tmp := tmpenv.New("GOFILE")
	defer tmp.Restore()
	os.Setenv("GOFILE", "foo.go")

	gen, err := trygo.NewGen(".")
	if err != nil {
		t.Fatal(err)
	}

	dirs, err := gen.PackageDirs([]string{})
	if err != nil {
		t.Fatal(err)
	}

	if len(dirs) != 1 || dirs[0] != cwd {
		t.Fatal("Current working dir must be set as package dir:", dirs)
	}
}

func TestGenerateFindPackageDirsError(t *testing.T) {
	for _, tc := range []struct {
		what  string
		paths []string
		want  string
	}{
		{
			what:  "no path",
			paths: []string{},
			want:  "not run from `go generate` and no path is given",
		},
		{
			what:  "directory not exist",
			paths: []string{"/path/to/unknown"},
			want:  "Cannot read directory",
		},
		{
			what:  "not a go package",
			paths: []string{filepath.FromSlash("testdata/gen/error/empty")},
			want:  "No Go package is included in given paths",
		},
	} {
		t.Run(tc.what, func(t *testing.T) {
			gen, err := trygo.NewGen(".")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := gen.PackageDirs(tc.paths); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatal("Unexpected error:", err)
			}
		})
	}
}

func TestGenerateNewGenError(t *testing.T) {
	_, err := trygo.NewGen("")
	if err == nil || !strings.Contains(err.Error(), "Output directory must be given") {
		t.Fatal("Unexpected error:", err)
	}
}
