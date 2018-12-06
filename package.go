package trygo

import (
	"bufio"
	"fmt"
	"github.com/pkg/errors"
	"go/ast"
	"go/format"
	"go/importer"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
)

// Package represents tranlated package. It contains tokens and AST of all Go files in the package
type Package struct {
	// Files is a token file set to get position information of nodes.
	Files *token.FileSet
	// Node is an AST package node which was parsed from TryGo code. AST will be directly modified
	// by translations.
	Node *ast.Package
	// Path is a package path where this translated package *will* be created.
	Path string
	// Birth is a pacakge path where this translated package was translated from.
	Birth string
	// Types is a type information of the package. This field is nil by default and set as the result
	// of verification. So this field is non-nil only when verification was performed.
	Types *types.Package
	// Flag which is set to true when AST is modified
	modified bool
}

func (pkg *Package) writeGo(out io.Writer, file *ast.File) error {
	w := bufio.NewWriter(out)
	if err := format.Node(w, pkg.Files, file); err != nil {
		if logEnabled {
			ast.Fprint(os.Stderr, pkg.Files, file, nil)
		}
		panic(fmt.Sprintf("Internal error: Broken Go source: %s: %s", file.Name.Name+".go", err))
	}
	return errors.Wrap(w.Flush(), "Cannot write file")
}

func (pkg *Package) writeGoFile(fpath string, file *ast.File) error {
	log("Write translated Go file to", hi(relpath(fpath)))

	if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
		return err
	}

	f, err := os.Create(fpath)
	if err != nil {
		return errors.Wrapf(err, "Cannot open output file %q", fpath)
	}
	defer f.Close()

	return pkg.writeGo(f, file)
}

func (pkg *Package) Write() error {
	log("Write translated package:", hi(pkg.Birth), "->", hi(pkg.Path))
	for path, node := range pkg.Node.Files {
		// Separate function to writeGoFile() to avoid `defer f.Close()` in loop
		if err := pkg.writeGoFile(path, node); err != nil {
			return err
		}
	}
	return nil
}

// WriteFileTo writes translated Go file to the given writer. If given file path does not indicate
// any translated source, it returns an error.
func (pkg *Package) WriteFileTo(out io.Writer, fpath string) error {
	f, ok := pkg.Node.Files[fpath]
	if !ok {
		return errors.Errorf("No file translated for %q", fpath)
	}
	return pkg.writeGo(out, f)
}

// Verify verifies the package is valid by type check. When there are some errors, it returns an error
// created by unifying all errors into one error.
func (pkg *Package) Verify() error {
	log("Verify translated package ", hi(pkg.Node.Name), "at", hi(relpath(pkg.Path)))
	// Verify translated package by type check
	errs := []error{}

	cfg := &types.Config{
		Importer:    importer.For("source", nil),
		FakeImportC: true,
		Error: func(err error) {
			log(ftl(err))
			errs = append(errs, err)
		},
	}

	files := make([]*ast.File, 0, len(pkg.Node.Files))
	for _, f := range pkg.Node.Files {
		files = append(files, f)
	}

	typeInfo, _ := cfg.Check(pkg.Path, pkg.Files, files, &types.Info{})
	if len(errs) > 0 {
		return unifyTypeErrors("verification after translation", errs)
	}
	pkg.Types = typeInfo

	// TODO: Add more verification for translation

	log("Package verification OK:", hi(pkg.Node.Name))
	return nil
}

// Modified returns the package was modified by translation
func (pkg *Package) Modified() bool {
	return pkg.modified
}

// Should add ParsePackage(pkgDir string, fs *token.FileSet) (*Package, error)?

// NewPackage creates a new Package instance containing additional information to AST node
func NewPackage(node *ast.Package, srcPath, destPath string, fs *token.FileSet) *Package {
	return &Package{
		Files:    fs,
		Node:     node,
		Path:     destPath,
		Birth:    srcPath,
		Types:    nil,
		modified: false,
	}
}
