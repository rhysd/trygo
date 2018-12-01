package trygo

import (
	"go/ast"
	"go/build"
	"strconv"
	"strings"
)

// Import statements which import translated packages are still looking wrong paths.
// Considering translation, the import paths must be fixed not to break compilation.

type importsFixer struct {
	transMap  map[string]string
	ctx       build.Context
	pathToDir map[string]string
	count     int
}

func (fixer *importsFixer) resolveImportPath(path string, pkgDir string) string {
	if p, ok := fixer.pathToDir[path]; ok {
		return p
	}
	p, err := fixer.ctx.Import(path, pkgDir, build.FindOnly)
	if err != nil {
		// Panic due to internal fatal error. Type check was already passed so import paths should
		// be resolved correctly.
		panic("Cannot resolve import path '" + path + "': " + err.Error())
	}
	fixer.pathToDir[path] = p.Dir
	log("Import path", hi(path), "was resolved to", hi(p.Dir))
	return p.Dir
}

func (fixer *importsFixer) fixImport(node *ast.ImportSpec, pkgDir string) {
	log("Looking import spec", hi(node.Path.Value))

	path, err := strconv.Unquote(node.Path.Value)
	if err != nil {
		// Panic due to internal fatal error. The AST node came from the parse results so literal value
		// must be correct Go expression.
		panic("Import path is broken Go string: " + node.Path.Value)
	}

	srcDir := fixer.resolveImportPath(path, pkgDir)

	destDir, ok := fixer.transMap[srcDir]
	if !ok {
		return
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
}

func (fixer *importsFixer) fixPackage(pkg *Package) {
	log("Fix imports:", hi(pkg.Node.Name))
	for fpath, file := range pkg.Node.Files {
		log("Fix imports in file:", hi(fpath))
		for _, node := range file.Imports {
			fixer.fixImport(node, pkg.Path)
		}
	}
}

func fixImports(pkgs []*Package) {
	l := len(pkgs)
	log("Fix imports in", l, "packages")
	m := make(map[string]string, l)
	for _, pkg := range pkgs {
		m[pkg.Birth] = pkg.Path
	}
	fixer := &importsFixer{m, build.Default, map[string]string{}, 0}
	for _, pkg := range pkgs {
		fixer.fixPackage(pkg)
	}
	log("Fix imports done.", fixer.count, "imports were fixed")
}