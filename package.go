package trygo

import (
	"bufio"
	"bytes"
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
	// Node is an AST package node which was parsed from TryGo code. AST will be directly modified by translations.
	Node *ast.Package
	// Path is a package path where this translated package *will* be created.
	Path string
	// Birth is a pacakge path where this translated package was translated from.
	Birth    string
	modified bool
}

func (pkg *Package) writeGoFileTo(out io.Writer, file *ast.File) error {
	w := bufio.NewWriter(out)
	if err := format.Node(w, pkg.Files, file); err != nil {
		var buf bytes.Buffer
		ast.Fprint(&buf, pkg.Files, file, nil)
		return errors.Wrapf(err, "Broken Go source: %s\n%s", file.Name.Name+".go", buf.String())
	}
	return errors.Wrap(w.Flush(), "Cannot write file")
}

func (pkg *Package) writeGoFile(fname string, file *ast.File) error {
	outpath := filepath.Join(pkg.Path, fname)
	log("Write translated Go file to", hi(relpath(fname)))

	if err := os.MkdirAll(filepath.Dir(outpath), 0755); err != nil {
		return err
	}

	f, err := os.Create(outpath)
	if err != nil {
		return errors.Wrapf(err, "Cannot open output file %q", outpath)
	}
	defer f.Close()

	return pkg.writeGoFileTo(f, file)
}

func (pkg *Package) Write() error {
	log("Write translated package:", hi(pkg.Birth), "->", hi(pkg.Path))
	for path, node := range pkg.Node.Files {
		fname := filepath.Base(path)
		if err := pkg.writeGoFile(fname, node); err != nil {
			return err
		}
	}
	return nil
}

func (pkg *Package) verify() error {
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

	cfg.Check(pkg.Path, pkg.Files, files, &types.Info{})
	if len(errs) > 0 {
		return unifyTypeErrors("verification after translation", errs)
	}

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
		modified: false,
	}
}
