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
	log("Collect package dir for `go generate`:", cwd)
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

// PackageDirs collects package directories under given paths. If paths argument is empty, it collects
// a package directory as `go generate` runs trygo. If no Go package is found or pacakge directory
// cannot be read, this function returns an error.
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

// TranslatePackages translates all packages specified with directory paths. It returns slice of Package
// which represent translated packages. When parsing Go(TryGo) sources failed or the translations failed,
// this function returns an error.
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
			parsed = append(parsed, NewPackage(pkg, dir, gen.outDirPath(dir), fset))
		}
	}

	// Translate all parsed ASTs per package
	if err := Translate(parsed); err != nil {
		return nil, err
	}

	return parsed, nil
}

// GeneratePackages translates all TryGo packages specified with directory paths and generates translated
// Go files with the same directory structures under output directory.
// When 'verify' argument is set to true, translated packages are verified with type checks after
// generating the Go files. When the verification reports some errors, generated Go files would be broken.
// This verification is mainly used for debugging.
// When parsing Go(TryGo) sources failed or the translations failed, translated Go file could not
// be written, this function returns an error.
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
			if !pkg.modified {
				log("Skip verification of unmodified package", pkg.Node.Name, "translated from", relpath(pkg.Birth))
				continue
			}
			if err := pkg.verify(); err != nil {
				return errors.Wrap(err, "Type error while verification after translation")
			}
		}
	}

	return nil
}

// Generate collects all TryGo packages under given paths, translates all the TryGo packages specified
// with directory paths and generates translated Go files with the same directory structures under
// output directory.
// When 'verify' argument is set to true, translated packages are verified with type checks after
// generating the Go files. When the verification reports some errors, generated Go files would be broken.
// This verification is mainly used for debugging.
// When collecting TryGo packages from paths failed, packages parsing TryGo sources failed or the translations
// failed, translated Go file could not be written, this function returns an error.
func (gen *Gen) Generate(paths []string, verify bool) error {
	log("Start translation and generation for", paths)

	dirs, err := gen.PackageDirs(paths)
	if err != nil {
		return err
	}
	log("Package directories:", hi(dirs))

	if err := os.MkdirAll(gen.OutDir, 0755); err != nil {
		return errors.Wrapf(err, "Cannot create output directory %q", gen.OutDir)
	}
	log("Created outdir:", hi(gen.OutDir))

	return gen.GeneratePackages(dirs, verify)
}

// NewGen creates a new Gen instance with given output directory. All translated packages are generated
// under the output directory. When the output directory does not exist, it is automatically created.
func NewGen(outDir string) (*Gen, error) {
	if outDir == "" {
		return nil, errors.New("Output directory must be given")
	}
	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(cwd, outDir)
	}
	return &Gen{outDir, os.Stdout}, nil
}
