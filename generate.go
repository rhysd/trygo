package trygo

import (
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
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

type Gen struct {
	OutDir string
	Writer io.Writer
}

func (gen *Gen) collectPackagesForGoGenerate() ([]Package, error) {
	if _, ok := os.LookupEnv("GOFILE"); !ok {
		return nil, errors.New("`trygo` was not run from `go generate` and no path is given. Nothing to generate")
	}

	fs, err := ioutil.ReadDir(cwd)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot read package directory %q", cwd)
	}

	names := []string{}
	for _, f := range fs {
		name := f.Name()
		if f.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		names = append(names, name)
	}

	// Note: Don't check `names` is empty since at least $GOFILE must exist
	pkgs := []Package{
		{cwd, names},
	}

	return pkgs, nil
}

func (gen *Gen) collectPackagesFromPaths(paths []string) ([]Package, error) {
	sawPkgs := map[string]*Package{}
	for _, path := range paths {
		if err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				sawPkgs[path] = &Package{Dir: path}
				return nil
			}
			if !strings.HasSuffix(p, ".go") {
				return nil
			}
			pkg := sawPkgs[filepath.Dir(path)]
			pkg.Sources = append(pkg.Sources, path)
			return nil
		}); err != nil {
			return nil, errors.Wrapf(err, "Cannot read directory %q", path)
		}
	}

	pkgs := make([]Package, 0, len(sawPkgs))
	for _, pkg := range sawPkgs {
		if len(pkg.Sources) > 0 {
			pkgs = append(pkgs, *pkg)
		}
	}

	if len(pkgs) == 0 {
		return nil, errors.New("No Go package is included in given paths")
	}

	return pkgs, nil
}

func (gen *Gen) CollectPackages(paths []string) ([]Package, error) {
	if len(paths) == 0 {
		return gen.collectPackagesForGoGenerate()
	}
	return gen.collectPackagesFromPaths(paths)
}

func (gen *Gen) GeneratePackages(pkgs []Package) error {
	// TODO: Create []*ast.File and *token.FileSet
	panic("TODO")
}

func (gen *Gen) Generate(paths []string) error {
	if err := os.MkdirAll(gen.OutDir, 0755); err != nil {
		return errors.Wrapf(err, "Cannot create output directory %q", gen.OutDir)
	}
	pkgs, err := gen.CollectPackages(paths)
	if err != nil {
		return err
	}
	return gen.GeneratePackages(pkgs)
}

func NewGen(outDir string) (*Gen, error) {
	if outDir == "" {
		return nil, errors.New("Output directory must be given")
	}
	return &Gen{outDir, os.Stdout}, nil
}
