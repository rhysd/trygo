package trygo

import (
	"fmt"
	"github.com/pkg/errors"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var cwd string

func init() {
	var err error
	if cwd, err = os.Getwd(); err != nil {
		panic(err)
	}
}

// Gen represents a generator of trygo
type Gen struct {
	// OutDir is a directory path to output directory. This value must be an absolute path
	OutDir string
	// Writer is a writer to output messages
	Writer io.Writer
}

func (gen *Gen) packageDirsForGoGenerate() ([]string, error) {
	if _, ok := os.LookupEnv("GOFILE"); !ok {
		return nil, errors.New("`trygo` was not run from `go generate` and no path is given. Nothing to generate")
	}
	log("Collect package dir for `go generate`")
	return []string{cwd}, nil
}

func (gen *Gen) packageDirsFromPaths(paths []string) ([]string, error) {
	log("Collect package dir for given paths:", hi(paths))

	saw := map[string]struct{}{}
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			path = filepath.Join(cwd, path)
		}
		if err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(p, ".go") {
				return nil
			}
			saw[filepath.Dir(p)] = struct{}{}
			return nil
		}); err != nil {
			return nil, errors.Wrapf(err, "Cannot read directory %q", path)
		}
	}

	l := len(saw)
	if l == 0 {
		return nil, errors.Errorf("No Go package is included in given paths: %v", paths)
	}

	dirs := make([]string, 0, l)
	for dir := range saw {
		dirs = append(dirs, dir)
	}

	return dirs, nil
}

func (gen *Gen) PackageDirs(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return gen.packageDirsForGoGenerate()
	}
	return gen.packageDirsFromPaths(paths)
}

func (gen *Gen) outDirPath(inpath string) string {
	// outDir: /repo/out
	// package: /repo/foo/bar

	d := gen.OutDir
	for !strings.HasPrefix(inpath, d) {
		d = filepath.Dir(d)
	}
	// d: /repo

	// part: /foo/bar
	part := strings.TrimPrefix(inpath, d)

	// return: /repo/out/foo/bar
	return filepath.Join(gen.OutDir, part)
}

func (gen *Gen) TranslatePackages(pkgDirs []string) ([]*Package, error) {
	log("Parse package directories:", pkgDirs)

	parsed := make([]*Package, 0, len(pkgDirs))
	fset := token.NewFileSet()
	for _, dir := range pkgDirs {
		pkgs, err := parser.ParseDir(fset, dir, nil, 0)
		if err != nil {
			return nil, err
		}
		for _, pkg := range pkgs {
			parsed = append(parsed, &Package{
				Files: fset,
				Node:  pkg,
				Path:  gen.outDirPath(dir),
				Birth: dir,
			})
		}
	}

	// Translate all parsed ASTs per package
	if err := Translate(parsed); err != nil {
		return nil, err
	}

	return parsed, nil
}

func (gen *Gen) GeneratePackages(pkgDirs []string, verify bool) error {
	pkgs, err := gen.TranslatePackages(pkgDirs)
	if err != nil {
		return err
	}
	log("Translation done:", len(pkgs), "packages")

	for _, pkg := range pkgs {
		if err := pkg.Write(); err != nil {
			return err
		}
		fmt.Fprintln(gen.Writer, pkg.Path)
	}

	if verify {
		for _, pkg := range pkgs {
			if err := pkg.verify(); err != nil {
				return errors.Wrap(err, "Type error while verification after translation")
			}
		}
	}

	return nil
}

// When verify is set to true, Gen verifies translated packages with type check again. This flag
// is mainly used for debugging
func (gen *Gen) Generate(paths []string, verify bool) error {
	log("Create outdir:", hi(gen.OutDir))
	if err := os.MkdirAll(gen.OutDir, 0755); err != nil {
		return errors.Wrapf(err, "Cannot create output directory %q", gen.OutDir)
	}

	dirs, err := gen.PackageDirs(paths)
	if err != nil {
		return err
	}
	log("Package directories:", hi(dirs))

	return gen.GeneratePackages(dirs, verify)
}

func NewGen(outDir string) (*Gen, error) {
	if outDir == "" {
		return nil, errors.New("Output directory must be given")
	}
	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(cwd, outDir)
	}
	return &Gen{outDir, os.Stdout}, nil
}
