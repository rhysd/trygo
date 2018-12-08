package trygo

import (
	"github.com/pkg/errors"
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
)

type transKind int

const (
	transKindInvalid transKind = iota
	transKindValueSpec
	transKindAssign
	transKindToplevelCall
	transKindExpr
)

func (kind transKind) String() string {
	switch kind {
	case transKindValueSpec:
		return "transValueSpec"
	case transKindAssign:
		return "transAssign"
	case transKindToplevelCall:
		return "transToplevelCall"
	case transKindExpr:
		return "transExpr"
	case transKindInvalid:
		return "transInvalid"
	default:
		panic("Unreachable")
	}
}

type transPoint struct {
	kind transKind
	// The target node. It must be one of *ast.ValueSpec, *ast.AssignStmt, *ast.ExprStmt, *ast.CallExpr.
	//   AssignStmt -> $vals, err = try(...) or $vals, err := try(...) (Depends on Tok field value)
	//   ValueStmt  -> var $vals, err = try(...)
	//   CallExpr   -> standalone try(...) call in general expressions
	//   ExprStmt   -> ExprStmt at toplevel of block
	node ast.Node
	// blockIndex is the index in list of statements at the block when this transPOint was created
	blockIndex int
	fun        ast.Node      // *ast.FuncDecl or *ast.FuncLit
	call       *ast.CallExpr // Function call in try() invocation
	parent     ast.Node
	pos        token.Pos
}

type blockTree struct {
	// This node can be ast.BlockStmt, ast.CaseClause, ast.CommClause
	ast ast.Stmt
	// transPoints *must* be in the order of statements in the block. Earlier statement must be before later statement in this slice.
	transPoints []*transPoint
	children    []*blockTree
	parent      *blockTree
}

func (tree *blockTree) stmts() []ast.Stmt {
	switch node := tree.ast.(type) {
	case *ast.BlockStmt:
		return node.List
	case *ast.CaseClause:
		return node.Body
	case *ast.CommClause:
		return node.Body
	default:
		panic("Unreachable")
	}
}

func (tree *blockTree) setStmts(stmts []ast.Stmt) {
	switch node := tree.ast.(type) {
	case *ast.BlockStmt:
		node.List = stmts
	case *ast.CaseClause:
		node.Body = stmts
	case *ast.CommClause:
		node.Body = stmts
	default:
		panic("Unreachable")
	}
}

// insertStmtAt inserts given statement *before* given index position of current block
func (tree *blockTree) insertStmtAt(idx int, stmt ast.Stmt) {
	logf("Insert %T statement at index %d of block %T", stmt, idx, tree.ast)
	prev := tree.stmts()
	l, r := prev[:idx], prev[idx:]
	ls := make([]ast.Stmt, 0, len(prev)+1)
	ls = append(ls, l...)
	ls = append(ls, stmt)
	ls = append(ls, r...)
	tree.setStmts(ls)
}

func (tree *blockTree) removeStmtAt(idx int) {
	prev := tree.stmts()
	logf("Remove %T statement at index %d of block %T", prev[idx], idx, tree.ast)
	l, r := prev[:idx], prev[idx+1:]
	tree.setStmts(append(l, r...))
}

func (tree *blockTree) isRoot() bool {
	return tree.parent == nil
}

func (tree *blockTree) collectTransPoints() []*transPoint {
	pts := tree.transPoints
	for _, c := range tree.children {
		if ps := c.collectTransPoints(); len(ps) > 0 {
			pts = append(pts, ps...)
		}
	}
	return pts
}

func unifyTypeErrors(phase string, errs []error) error {
	l := len(errs)
	var b strings.Builder
	b.WriteString("Type error(s) at ")
	b.WriteString(phase)
	b.WriteString(":\n")
	for i, err := range errs {
		b.WriteString("  ")
		b.WriteString(err.Error())
		if i < l-1 {
			b.WriteRune('\n')
		}
	}
	return errors.New(b.String())
}

func typeCheck(transPts []*transPoint, pkgDir string, fset *token.FileSet, files []*ast.File) (*types.Info, *types.Package, error) {
	errs := []error{}
	cfg := &types.Config{
		Importer:    importer.For("source", nil),
		FakeImportC: true,
		Error: func(err error) {
			log(ftl(err))
			errs = append(errs, err)
		},
	}

	tys := map[ast.Expr]types.TypeAndValue{}
	for _, trans := range transPts {
		if lit, ok := trans.fun.(*ast.FuncLit); ok {
			// For getting the return type of function for building zero values at if err != nil check body
			tys[lit] = types.TypeAndValue{}
		}
		if trans.kind == transKindToplevelCall {
			// For getting the return type of try(f(..)) at *ast.ExprStmt
			tys[trans.call] = types.TypeAndValue{}
		}
	}

	info := &types.Info{
		Types: tys,
		Defs:  map[*ast.Ident]types.Object{},
	}

	pkg, _ := cfg.Check(pkgDir, fset, files, info)
	if len(errs) > 0 {
		return nil, nil, unifyTypeErrors("type check after phase-1", errs)
	}

	if logEnabled {
		var b strings.Builder
		b.WriteString(hi("Types for identifiers: "))
		for ident, obj := range info.Defs {
			b.WriteString(hi(ident.Name))
			if obj != nil {
				b.WriteString(":'" + obj.String() + "'")
			}
			b.WriteString(", ")
		}
		log(b.String())
		log(hi("Types for package "+pkg.Name()), pkg.String())
	}

	return info, pkg, nil
}

// translatePackage translates given package from TryGo to Go. Given AST is directly modified. When error
// occurs, it returns an error and the AST may be incompletely modified.
func translatePackage(pkg *Package) error {
	pkgName := pkg.Node.Name
	log("Translation", hi("start: "+pkgName))

	tce := &tryCallElimination{
		pkg:     pkg.Node,
		fileset: pkg.Files,
	}

	log(hi("Phase-1"), "try() call elimination", hi("start: "+pkgName))
	// Traverse AST for phase-1
	ast.Walk(tce, pkg.Node)
	if tce.err != nil {
		return tce.err
	}
	tce.assertPostCondition()
	log(hi("Phase-1"), "try() call elimination", hi("end: "+pkgName))

	log("Number of translations:", hi(tce.numTrans))
	if tce.numTrans == 0 {
		// Nothing was translated. Can skip later process
		return nil
	}

	log(hi("Type check"), "after phase-1", hi("start: "+pkgName))
	files := make([]*ast.File, 0, len(pkg.Node.Files))
	for _, f := range pkg.Node.Files {
		files = append(files, f)
	}

	transPoints := []*transPoint{}
	for _, root := range tce.roots {
		transPoints = append(transPoints, root.collectTransPoints()...)
	}

	tyInfo, tyPkg, err := typeCheck(transPoints, pkg.Birth, pkg.Files, files)
	if err != nil {
		// TODO: More informational error. Which translation failed? Is it related to try() elimination? Or simply original code has type error?
		log(ftl(err))
		return err
	}
	log(hi("Type check"), "after phase-1", hi("end: "+pkgName))

	nci := &nilCheckInsertion{
		pkg:      pkg.Node,
		fileset:  pkg.Files,
		roots:    tce.roots,
		typeInfo: tyInfo,
		pkgTypes: tyPkg,
	}

	// Traverse blocks for phase-2
	log(hi("Phase-2"), "if err != nil check insertion", hi("start: "+pkgName))
	nci.translate()
	log(hi("Phase-2"), "if err != nil check insertion", hi("end: "+pkgName))

	log("Translation", hi("end: "+pkgName))
	pkg.modified = true
	return nil
}

// Translate translates all given TryGo packages by modifying given slice of Packages directly. After
// translation, the given packages are translated to Go packages.
// Each Package instance's Node and Files fields must be set with the results of an AST and tokens
// parsed from TryGo source. And Birth must be set correctly as package directory of the TryGo source.
// When translation failed, it returns an error as soon as possible. Given Package instances may be
// no longer correct.
func Translate(pkgs []*Package) error {
	log("Translate parsed packages:", pkgs)

	// Translate try() calls with 2 stages
	for _, pkg := range pkgs {
		if err := translatePackage(pkg); err != nil {
			return errors.Wrapf(err, "While translating %s", pkg.Birth)
		}
	}

	// Fix all import paths considering translations
	if err := fixImports(pkgs); err != nil {
		return err
	}

	// Fix file paths considering translations
	for _, pkg := range pkgs {
		files := make(map[string]*ast.File, len(pkg.Node.Files))
		for path, file := range pkg.Node.Files {
			outpath := filepath.Join(pkg.Path, filepath.Base(path))
			files[outpath] = file
		}
		pkg.Node.Files = files
	}

	if logEnabled {
		modified := make([]string, 0, len(pkgs))
		for _, pkg := range pkgs {
			if pkg.modified {
				modified = append(modified, pkg.Node.Name)
			}
		}
		log("Translation done. Total packages:", hi(len(pkgs)), "Modified packages:", hi(len(modified)), modified)
	}
	return nil
}
