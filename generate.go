package trygo

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"go/ast"
	"go/format"
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

func (gen *Gen) outFilePath(inpath string) string {
	// outDir: /repo/out
	// package: /repo/foo/bar

	d := gen.OutDir
	for !strings.HasPrefix(inpath, d) {
		d = filepath.Dir(d)
	}
	// d: /repo

	// part: /foo/bar
	part := strings.TrimPrefix(inpath, d)

	// return: repo/out/foo/bar
	return filepath.Join(gen.OutDir, part)
}

func (gen *Gen) writeGo(out io.Writer, file *ast.File, fset *token.FileSet) error {
	w := bufio.NewWriter(out)
	if err := format.Node(w, fset, file); err != nil {
		var buf bytes.Buffer
		ast.Fprint(&buf, fset, file, nil)
		return errors.Wrapf(err, "Broken Go source: %s\n%s", file.Name.Name+".go", buf.String())
	}
	return w.Flush()
}

func (gen *Gen) writeGoFile(path string, file *ast.File, fset *token.FileSet) error {
	outpath := gen.outFilePath(path)
	log("Write translated file:", hi(relpath(path)), "->", hi(relpath(outpath)))

	if err := os.MkdirAll(filepath.Dir(outpath), 0755); err != nil {
		return err
	}

	outfile, err := os.Create(outpath)
	if err != nil {
		return errors.Wrapf(err, "Cannot open output file %q", outpath)
	}
	defer outfile.Close()

	if err := gen.writeGo(outfile, file, fset); err != nil {
		return err
	}

	fmt.Fprintln(gen.Writer, outpath)
	return nil
}

func (gen *Gen) TranslatePackages(pkgDirs []string) (map[string]*ast.Package, *token.FileSet, error) {
	log("Parse package directories:", pkgDirs)

	parsed := make(map[string]*ast.Package, len(pkgDirs))
	fset := token.NewFileSet()
	for _, dir := range pkgDirs {
		pkgs, err := parser.ParseDir(fset, dir, nil, 0)
		if err != nil {
			return nil, nil, err
		}
		for dir, pkg := range pkgs {
			parsed[dir] = pkg
		}
	}

	// Translate all parsed ASTs per package
	log("Translate parsed packages:", parsed)
	for dir, pkg := range parsed {
		if err := Translate(dir, pkg, fset); err != nil {
			return nil, nil, err
		}
	}

	return parsed, fset, nil
}

func (gen *Gen) GeneratePackages(pkgDirs []string) error {
	pkgs, fset, err := gen.TranslatePackages(pkgDirs)
	if err != nil {
		return err
	}

	log("Write translated packages to files:", pkgs)
	for _, pkg := range pkgs {
		for path, ast := range pkg.Files {
			if err := gen.writeGoFile(path, ast, fset); err != nil {
				return err
			}
		}
	}

	return nil
}

func (gen *Gen) Generate(paths []string) error {
	log("Create outdir:", hi(gen.OutDir))
	if err := os.MkdirAll(gen.OutDir, 0755); err != nil {
		return errors.Wrapf(err, "Cannot create output directory %q", gen.OutDir)
	}

	dirs, err := gen.PackageDirs(paths)
	if err != nil {
		return err
	}
	log("Package directories:", hi(dirs))

	return gen.GeneratePackages(dirs)
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
