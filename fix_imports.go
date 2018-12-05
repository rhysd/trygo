package trygo

import (
	"fmt"
	"github.com/pkg/errors"
	"go/ast"
	"go/build"
	"strconv"
	"strings"
)

// Import statements which import translated packages are still looking wrong paths.
// Considering translation, the import paths must be fixed not to break compilation.

type importError struct {
	msg  string
	node ast.Node
}

func (err *importError) Error() string {
	return err.msg
}

type importsFixer struct {
	transMap  map[string]string
	ctx       build.Context
	pathToDir map[string]string
	count     int
	errs      []*importError
}

func (fixer *importsFixer) errAt(node ast.Node, msg string) {
	err := &importError{msg, node}
	log(ftl(err))
	fixer.errs = append(fixer.errs, err)
}

func (fixer *importsFixer) errfAt(node ast.Node, format string, args ...interface{}) {
	fixer.errAt(node, fmt.Sprintf(format, args...))
}

func (fixer *importsFixer) resolveImportPath(path string, pkgDir string) (string, error) {
	if p, ok := fixer.pathToDir[path]; ok {
		return p, nil
	}
	p, err := fixer.ctx.Import(path, pkgDir, build.FindOnly)
	if err != nil {
		return "", err
	}
	fixer.pathToDir[path] = p.Dir
	log("Import path", hi(path), "was resolved to", hi(p.Dir))
	return p.Dir, nil
}

func (fixer *importsFixer) fixImport(node *ast.ImportSpec, pkgDir string) bool {
	log("Looking import spec", hi(node.Path.Value))

	path, err := strconv.Unquote(node.Path.Value)
	if err != nil {
		// Panic due to internal fatal error. The AST node came from the parse results so literal value
		// must be correct Go expression.
		panic("Import path is broken Go string: " + node.Path.Value)
	}

	srcDir, err := fixer.resolveImportPath(path, pkgDir)
	if err != nil {
		// This error may happen in normal case when translating TryGo code does not contain any try() call.
		// In the case, try() call elimination returns early just after stage 1 and type check is not performed.
		// When the TryGo code contains an invalid import statement, it causes an error here.
		fixer.errfAt(node, "Cannot resolve import path %q: %s", path, err)
		return false
	}

	destDir, ok := fixer.transMap[srcDir]
	if !ok {
		return false
	}

	// path: trygo/some/pkg
	// srcDir: /path/to/trygo/some/pkg
	// destDir: /path/to/outdir/some/pkg

	// prefix: /path/to/
	prefix := strings.TrimSuffix(srcDir, path)

	// transPath: outdir/some/pkg
	transPath := strings.TrimPrefix(destDir, prefix)

	// Finally replace import path with translated directory
	prev := node.Path.Value
	node.Path.Value = strconv.Quote(transPath)
	log("Fix imoprt path:", hi(prev), "->", hi(node.Path.Value))
	fixer.count++
	return true
}

func (fixer *importsFixer) fixPackage(pkg *Package) {
	log("Fix imports:", hi(pkg.Node.Name))
	for fpath, file := range pkg.Node.Files {
		log("Fix imports in file:", hi(fpath))
		for _, node := range file.Imports {
			if fixer.fixImport(node, pkg.Path) {
				pkg.modified = true
			}
		}
	}
}

func fixImports(pkgs []*Package) error {
	l := len(pkgs)
	log("Fix imports in", l, "packages")
	m := make(map[string]string, l)
	for _, pkg := range pkgs {
		m[pkg.Birth] = pkg.Path
	}

	fixer := &importsFixer{m, build.Default, map[string]string{}, 0, nil}
	for _, pkg := range pkgs {
		fixer.fixPackage(pkg)
	}

	if len(fixer.errs) > 0 {
		fset := pkgs[0].Files
		if len(fixer.errs) == 1 {
			pos := fset.Position(fixer.errs[0].node.Pos())
			err := errors.Errorf("Import error while fixing import paths: At %s: %s", pos, fixer.errs[0])
			log(ftl(err))
			return err
		}

		var b strings.Builder
		fmt.Fprintf(&b, "%d import error(s) while fixing import paths:", len(fixer.errs))
		for _, err := range fixer.errs {
			fmt.Fprintf(&b, "\n  %s at %s", err.msg, fset.Position(err.node.Pos()))
		}

		msg := b.String()
		log(ftl(msg))
		return errors.New(msg)
	}

	log("Fix imports done.", fixer.count, "imports were fixed")
	return nil
}
