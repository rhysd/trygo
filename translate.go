package trygo

import (
	"fmt"
	"github.com/pkg/errors"
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
	"reflect"
	"strings"
)

type nodeStack []ast.Node

func (ns nodeStack) push(n ast.Node) nodeStack {
	return append(ns, n)
}
func (ns nodeStack) pop() nodeStack {
	return ns[:len(ns)-1]
}
func (ns nodeStack) top() ast.Node {
	return ns[len(ns)-1]
}
func (ns nodeStack) show() string {
	var b strings.Builder
	b.WriteString("BOTTOM <- ")
	for _, n := range ns {
		b.WriteString(reflect.TypeOf(n).String())
		b.WriteString(" <- ")
	}
	b.WriteString("TOP")
	return b.String()
}
func (ns nodeStack) assertEmpty(forWhat string) {
	if len(ns) == 0 {
		return
	}
	panic(fmt.Sprintf("AST node stack for %s is not fully poped: %s", forWhat, ns.show()))
}

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
		return "TransValueSpec"
	case transKindAssign:
		return "TransAssign"
	case transKindToplevelCall:
		return "TransToplevelCall"
	case transKindExpr:
		return "TransExpr"
	case transKindInvalid:
		return "TransInvalid"
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
	node       ast.Node
	block      *ast.BlockStmt
	blockIndex int
	Func       ast.Node      // *ast.FuncDecl or *ast.FuncLit
	call       *ast.CallExpr // Function call in try() invocation
	parent     ast.Node
	pos        token.Pos
}

func (tp *transPoint) funcType() *ast.FuncType {
	switch f := tp.Func.(type) {
	case *ast.FuncLit:
		return f.Type
	case *ast.FuncDecl:
		return f.Type
	default:
		return nil
	}
}

type blockTree struct {
	ast *ast.BlockStmt
	// transPoints *must* be in the order of statements in the block. Earlier statement must be before later statement in this slice.
	transPoints []*transPoint
	children    []*blockTree
	parent      *blockTree
}

func (tree *blockTree) isRoot() bool {
	return tree.parent == nil
}

type tryCallElimination struct {
	pkg        *ast.Package
	fileset    *token.FileSet
	err        error
	file       *ast.File
	roots      []*blockTree
	parentBlk  *blockTree
	currentBlk *blockTree
	blkIndex   int
	parents    nodeStack
	funcs      nodeStack
	numTrans   int
}

func (tce *tryCallElimination) assertPostCondition() {
	tce.parents.assertEmpty("parents")
	tce.funcs.assertEmpty("funcs")
	if tce.parentBlk != nil || tce.currentBlk != nil {
		panic(fmt.Sprintf("Parent block and/or current block are not nil. parent:%v current:%v", tce.parentBlk, tce.currentBlk))
	}
}

func (tce *tryCallElimination) errAt(node ast.Node, msg string) {
	pos := tce.fileset.Position(node.Pos())
	tce.err = errors.Errorf("%s: %v: Error: %s", pos, tce.pkg.Name, msg)
	log(ftl(tce.err))
}

func (tce *tryCallElimination) errfAt(node ast.Node, format string, args ...interface{}) {
	tce.errAt(node, fmt.Sprintf(format, args...))
}

// checkTryCall checks given try() call and returns try() call and inner call (the argument of the try call)
// since try()'s argument must be function call. When it is not a try() call, it returns nil as first the
// return value. When it is an invalid try() call, it sets the error to err field and returns false
// as the third return value.
func (tce *tryCallElimination) checkTryCall(maybeCall ast.Expr) (tryCall *ast.CallExpr, innerCall *ast.CallExpr, ok bool) {
	outer, ok := maybeCall.(*ast.CallExpr)
	if !ok {
		log("Skipped since expression is not a call expression")
		return nil, nil, true
	}

	name, ok := outer.Fun.(*ast.Ident)
	if !ok {
		log("Skipped since callee was not var ref")
		return nil, nil, true
	}
	if name.Name != "try" {
		log("Skipped since RHS is not calling 'try':", name.Name)
		return nil, nil, true
	}

	if len(outer.Args) != 1 {
		tce.errfAt(outer, "try() takes 1 argument but %d found", len(outer.Args))
		return nil, nil, false
	}

	inner, ok := outer.Args[0].(*ast.CallExpr)
	if !ok {
		tce.errfAt(outer, "try() call's argument must be function call but found %s", reflect.TypeOf(outer.Args[0]))
		return nil, nil, false
	}

	if len(tce.funcs) == 0 {
		tce.errAt(outer, "try() function is used outside function")
		return nil, nil, false
	}

	var funcTy *ast.FuncType
	switch f := tce.funcs.top().(type) {
	case *ast.FuncLit:
		funcTy = f.Type
	case *ast.FuncDecl:
		funcTy = f.Type
	}
	rets := funcTy.Results.List
	if len(rets) == 0 {
		tce.errAt(outer, "The function returns nothing. try() is not available")
		return nil, nil, false
	}

	ty := rets[len(rets)-1].Type
	if ident, ok := ty.(*ast.Ident); !ok || ident.Name != "error" {
		tce.errfAt(outer, "The function does not return error as last return value. Last return type is %q", ty)
		return nil, nil, false
	}

	log(hi("try() found:"), inner.Fun)
	return outer, inner, true
}

func (tce *tryCallElimination) eliminateTryCall(kind transKind, node ast.Node, maybeTryCall ast.Expr) bool {
	tryCall, innerCall, ok := tce.checkTryCall(maybeTryCall)
	if !ok || tryCall == nil {
		log("Skipped since the function call is not try() call or invalid try() call")
		return false
	}

	pos := tryCall.Pos()
	log(hi("Eliminate try() call"), "for kind", kind, "at", tce.fileset.Position(pos))

	// Squash try() call with inner call: try(f(...)) -> f(...)
	*tryCall = *innerCall

	p := &transPoint{
		kind:       kind,
		node:       node,
		block:      tce.currentBlk.ast,
		blockIndex: tce.blkIndex,
		Func:       tce.funcs.top(),
		call:       tryCall, // tryCall points inner call here
		parent:     tce.parents.top(),
		pos:        pos,
	}
	tce.currentBlk.transPoints = append(tce.currentBlk.transPoints, p)

	log("New TransPoint was added. Now size of points is", len(tce.currentBlk.transPoints))
	tce.numTrans++

	return true
}

func (tce *tryCallElimination) visitSpec(spec *ast.ValueSpec) {
	pos := tce.fileset.Position(spec.Pos())
	log("Value spec at", pos)

	if len(spec.Values) != 1 {
		// In Go, multiple LHS expressions means they does not return multiple values
		// Note: Following is ill-formed:
		//   var fromF = F(), try(funcOnlyReturnErr())
		log("Skipped due to multiple RHS values")
		return
	}

	if ok := tce.eliminateTryCall(transKindValueSpec, spec, spec.Values[0]); !ok {
		return
	}

	// Not to break type check, add _ at LHS here
	//
	// Total translation at stage-1 is:
	//   From:
	//     var $retvals = try(f(...))
	//   To:
	//     $retvals, _ = f(...)
	spec.Names = append(spec.Names, ast.NewIdent("_"))

	log("Value spec at", pos, "added new translation point")
}

func (tce *tryCallElimination) visitAssign(assign *ast.AssignStmt) {
	pos := tce.fileset.Position(assign.Pos())
	log("Assignment at", pos)

	if len(assign.Rhs) != 1 {
		// In Go, multiple LHS expressions means they does not return multiple values
		// Note: Following is ill-formed:
		//   fromF := F(), try(funcOnlyReturnErr())
		log("Skipped due to multiple RHS values")
		return
	}

	if ok := tce.eliminateTryCall(transKindAssign, assign, assign.Rhs[0]); !ok {
		return
	}

	// Not to break type check, add _ at LHS here
	//
	// Total translation at stage-1 is:
	//   From:
	//     $retvals := try(f(...))
	//   To:
	//     $retvals, _ := f(...)
	//
	//   From:
	//     $retvals = try(f(...))
	//   To:
	//     $retvals, _ = f(...)
	assign.Lhs = append(assign.Lhs, ast.NewIdent("_"))

	log("Assignment at", pos, "added new translation point")
}

func (tce *tryCallElimination) visitToplevelExpr(stmt *ast.ExprStmt) {
	pos := tce.fileset.Position(stmt.Pos())
	log("Assignment at", pos)

	expr := stmt.X
	if ok := tce.eliminateTryCall(transKindValueSpec, stmt, expr); !ok {
		return
	}

	log("Assignment at", pos, "added new translation point")
}

func (tce *tryCallElimination) visitBlock(block *ast.BlockStmt) {
	pos := tce.fileset.Position(block.Pos()).String()
	log("Block statement start", pos)

	tree := &blockTree{ast: block, parent: tce.parentBlk}
	if tree.isRoot() {
		if tce.currentBlk != nil {
			panic("FATAL: Root tree is not set after visiting block at " + tce.fileset.Position(tce.currentBlk.ast.Pos()).String())
		}
		log("New root block added at", pos)
		tce.roots = append(tce.roots, tree)
	} else {
		tce.parentBlk.children = append(tce.parentBlk.children, tree)
	}

	tce.parentBlk = tce.currentBlk
	tce.currentBlk = tree
	prevIdx := tce.blkIndex

	for i, stmt := range block.List {
		if tce.err != nil {
			return
		}

		tce.blkIndex = i

		if e, ok := stmt.(*ast.ExprStmt); ok {
			tce.visitToplevelExpr(e)
		} else {
			// Recursively visit
			ast.Walk(tce, stmt)
		}
	}

	tce.blkIndex = prevIdx
	tce.currentBlk = tce.parentBlk
	if !tree.isRoot() {
		tce.parentBlk = tce.parentBlk.parent
	}
	log("Block statement end", pos)
}

func (tce *tryCallElimination) visitPre(node ast.Node) ast.Visitor {
	switch node := node.(type) {
	case *ast.BlockStmt:
		tce.visitBlock(node)
		return nil // visitBlock recursively calls ast.Walk()
	case *ast.ValueSpec:
		// var or const
		tce.visitSpec(node)
	case *ast.AssignStmt:
		// := or =
		tce.visitAssign(node)
	case *ast.FuncDecl:
		tce.funcs = tce.funcs.push(node)
		log("Start function:", hi(node.Name.Name))
	case *ast.FuncLit:
		tce.funcs = tce.funcs.push(node)
		log("Start function literal")
	case *ast.File:
		log("File:", hi(node.Name.Name+".go"))
		tce.file = node
	}
	return tce
}

func (tce *tryCallElimination) visitPost(node ast.Node) {
	switch node := node.(type) {
	case *ast.FuncDecl:
		tce.funcs = tce.funcs.pop()
		log("End function:", hi(node.Name.Name))
	case *ast.FuncLit:
		tce.funcs = tce.funcs.pop()
		log("End function literal")
	}
}

func (tce *tryCallElimination) Visit(node ast.Node) ast.Visitor {
	if tce.err != nil {
		return nil
	}

	if node == nil {
		n := tce.parents.top()
		tce.parents = tce.parents.pop()
		tce.visitPost(n)
		return nil
	}

	v := tce.visitPre(node)
	if v != nil {
		// If return value is nil, it means that it will not visit children recursively. It means
		// that tce.VisitPre() visits its children by itself. In the case, pushing the node to parents
		// stack pushes the same node twice.
		tce.parents = tce.parents.push(node)
	}

	// When no error occurred, always visit children. Stopping visiting children collapses parents stack.
	// Note: It may be OK to return nil here. When return value is nil, we would also need to pop parents stack.
	return v
}

func typeCheck(pkgDir string, fset *token.FileSet, files []*ast.File) (*types.Info, *types.Package, error) {
	errs := []error{}
	cfg := &types.Config{
		Importer:    importer.Default(),
		FakeImportC: true,
		Error: func(err error) {
			log(ftl(err))
			errs = append(errs, err)
		},
	}
	info := &types.Info{
		// TODO: Add 'Types' field to collect type information of interesting nodes
		Defs: map[*ast.Ident]types.Object{},
	}

	pkg, _ := cfg.Check(pkgDir, fset, files, info)
	if len(errs) > 0 {
		var b strings.Builder
		b.WriteString("Type error(s):\n")
		for _, err := range errs {
			b.WriteString("  ")
			b.WriteString(err.Error())
			b.WriteRune('\n')
		}
		return nil, nil, errors.New(b.String())
	}

	// TODO: Check $CallExpr as argument of try() returns error as the last return value

	if logEnabled {
		var b strings.Builder
		b.WriteString(hi("Types for identifiers: "))
		for ident, obj := range info.Defs {
			b.WriteString(ident.Name)
			b.WriteRune(':')
			if obj == nil {
				b.WriteString("nil")
			} else {
				b.WriteString(obj.String())
			}
			b.WriteString(", ")
		}
		log(b.String())
		log(hi("Types for package "+pkg.Name()), pkg.String())
	}

	return info, pkg, nil
}

type nilCheckInsertion struct {
	pkg      *ast.Package
	fileset  *token.FileSet
	roots    []*blockTree
	blk      *ast.BlockStmt
	offset   int
	varID    int
	typeInfo *types.Info
	pkgTypes *types.Package
}

func (nci *nilCheckInsertion) genIdent() *ast.Ident {
	id := nci.varID
	nci.varID++
	return ast.NewIdent(fmt.Sprintf("_%d", id))
}

func (nci *nilCheckInsertion) genErrIdent() *ast.Ident {
	id := nci.varID
	nci.varID++
	return ast.NewIdent(fmt.Sprintf("_err%d", id))
}

// insertStmts inserts given statements *before* given index position of current block. If previous
// translation exists in the same block and some statements were already inserted, the offset is
// automatically adjusted.
func (nci *nilCheckInsertion) insertStmtsAt(idx int, stmts []ast.Stmt) {
	prev := nci.blk.List
	idx += nci.offset
	l, r := prev[:idx], prev[idx:]
	ls := make([]ast.Stmt, 0, len(prev)+len(stmts))
	ls = append(ls, l...)
	ls = append(ls, stmts...)
	ls = append(ls, r...)
	nci.blk.List = ls
	nci.offset += len(stmts)
	log(hi(len(stmts)), "statements inserted to block at", nci.fileset.Position(nci.blk.Pos()))
}

func (nci *nilCheckInsertion) removeStmtAt(idx int) {
	prev := nci.blk.List
	idx += nci.offset
	l, r := prev[:idx], prev[idx+1:]
	nci.blk.List = append(l, r...)
	nci.offset--
	log(hi(idx+1, "th statement was removed from block at", nci.fileset.Position(nci.blk.Pos())))
}

// TODO: Take type information of zero values to return in 'if' statement body instead of numRetVals
func (nci *nilCheckInsertion) insertIfNilChkStmtAfter(index int, errIdent *ast.Ident, init ast.Stmt, numRetVals int) {
	// TODO: Create correct zero values
	retVals := make([]ast.Expr, 0, numRetVals)
	for i := 0; i < numRetVals-1; i++ {
		retVals = append(retVals, ast.NewIdent("nil"))
	}
	retVals = append(retVals, errIdent)

	stmt := &ast.IfStmt{
		Init: init,
		Cond: &ast.BinaryExpr{
			X:  errIdent,
			Y:  ast.NewIdent("nil"),
			Op: token.NEQ,
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: retVals,
				},
			},
		},
	}

	nci.insertStmtsAt(index+1, []ast.Stmt{stmt})
	log("Inserted `if` statement for nil check at index", index+1, "of block at", nci.fileset.Position(nci.blk.Pos()))
}

func (nci *nilCheckInsertion) insertIfNilChkExprAt(index int, call *ast.CallExpr, numRetVals int) {
	log("Inserting if err := $call; err != nil { ... } at index", index, "of block at", nci.fileset.Position(nci.blk.Pos()))
	nci.removeStmtAt(index)

	errIdent := ast.NewIdent("err")
	assign := &ast.AssignStmt{
		Lhs: []ast.Expr{errIdent},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{call},
	}

	nci.insertIfNilChkStmtAfter(index, errIdent, assign, numRetVals)
}

func (nci *nilCheckInsertion) transValueSpec(node *ast.ValueSpec, trans *transPoint) {
	// From:
	//   var $retvals, _ = f(...)
	// To:
	//   var $retvals, err = f(...)
	//   if err != nil {
	//     return $zerovals, err
	//   }
	errIdent := ast.NewIdent("err")
	node.Names[len(node.Names)-1] = errIdent
	nci.insertIfNilChkStmtAfter(trans.blockIndex, errIdent, nil, trans.funcType().Results.NumFields())
}

func (nci *nilCheckInsertion) transAssign(node *ast.AssignStmt, trans *transPoint) {
	// From:
	//   $retvals, _ := f(...)
	// To:
	//   $retvals, err := f(...)
	//   if err != nil {
	//     return $zerovals, err
	//   }
	if node.Tok == token.DEFINE {
		log("Define statement(:=) is translated")
		errIdent := ast.NewIdent("err")
		node.Lhs[len(node.Lhs)-1] = errIdent
		nci.insertIfNilChkStmtAfter(trans.blockIndex, errIdent, nil, trans.funcType().Results.NumFields())
		return
	}

	// From:
	//   $retvals, _ = f(...)
	// To:
	//   var _err$n error
	//   $retvals, _err$n = f(...)
	//   if _err$n != nil {
	//     return $zerovals, _err$n
	//   }
	// Tok is token.EQ
	errIdent := nci.genErrIdent()
	log("Assign statement(=) is translated", hi(errIdent.Name))
	decl := &ast.DeclStmt{
		Decl: &ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{
				&ast.ValueSpec{
					Names: []*ast.Ident{
						errIdent,
					},
					Type: ast.NewIdent("error"),
				},
			},
		},
	}
	// Insert `var _err$n error`
	nci.insertStmtsAt(trans.blockIndex, []ast.Stmt{decl})

	node.Lhs[len(node.Lhs)-1] = errIdent
	nci.insertIfNilChkStmtAfter(trans.blockIndex, errIdent, nil, trans.funcType().Results.NumFields())
}

func (nci *nilCheckInsertion) transToplevelExpr(trans *transPoint) {
	// From:
	//   f(...)
	// To:
	//   if $ignores, err := f(...); err != nil {
	//     return $zerovals, err
	//   }
	nci.insertIfNilChkExprAt(trans.blockIndex, trans.call, trans.funcType().Results.NumFields())
}

func (nci *nilCheckInsertion) insertNilCheck(trans *transPoint) error {
	pos := nci.fileset.Position(trans.pos)
	log(hi("Insert if err != nil check for "+trans.kind.String()), "at", pos)

	switch trans.kind {
	case transKindValueSpec:
		nci.transValueSpec(trans.node.(*ast.ValueSpec), trans)
	case transKindAssign:
		nci.transAssign(trans.node.(*ast.AssignStmt), trans)
	case transKindToplevelCall:
		nci.transToplevelExpr(trans)
	case transKindExpr:
		panic("TODO")
	default:
		panic("Unreachable")
	}

	return nil
}

func (nci *nilCheckInsertion) block(b *blockTree) error {
	nci.blk = b.ast
	nci.offset = 0
	nci.varID = 0

	pos := nci.fileset.Position(b.ast.Pos())
	log("Start nil check insertion for block at", pos)
	for _, trans := range b.transPoints {
		if err := nci.insertNilCheck(trans); err != nil {
			return err
		}
	}
	log("End nil check insertion for block at", pos)

	log("Recursively insert nil check to", hi(len(b.children)), "children")
	for _, child := range b.children {
		if err := nci.block(child); err != nil {
			return err
		}
	}

	return nil
}

func (nci *nilCheckInsertion) translate() error {
	for _, root := range nci.roots {
		if err := nci.block(root); err != nil {
			return err
		}
	}
	return nil
}

// Translate translates given package from TryGo to Go. Given AST is directly modified. When error
// occurs, it returns an error and the AST may be incompletely modified.
func Translate(pkgDir string, pkg *ast.Package, fs *token.FileSet) error {
	pkgName := pkg.Name
	log("Translation", hi("start: "+pkgName))

	tce := &tryCallElimination{
		pkg:     pkg,
		fileset: fs,
	}

	log(hi("Phase-1"), "try() call elimination", hi("start: "+pkgName))
	// Traverse AST for phase-1
	ast.Walk(tce, pkg)
	if tce.err != nil {
		return tce.err
	}
	tce.assertPostCondition()
	log(hi("Phase-1"), "try() call elimination", hi("end: "+pkgName))

	if tce.numTrans == 0 {
		// Nothing was translated. Can skip later process
		return nil
	}

	log(hi("Type check"), "after phase-1", hi("start: "+pkgName))
	// TODO: Check types for files which only require translations (contain one or more `try()` calls)
	files := make([]*ast.File, 0, len(pkg.Files))
	for _, f := range pkg.Files {
		files = append(files, f)
	}
	tyInfo, tyPkg, err := typeCheck(pkgDir, fs, files)
	if err != nil {
		// TODO: More informational error. Which translation failed? Is it related to try() elimination? Or simply original code has type error?
		log(ftl(err))
		return err
	}
	log(hi("Type check"), "after phase-1", hi("end: "+pkgName))

	nci := &nilCheckInsertion{
		pkg:      pkg,
		fileset:  fs,
		roots:    tce.roots,
		typeInfo: tyInfo,
		pkgTypes: tyPkg,
	}

	log(hi("Phase-2"), "if err != nil check insertion", hi("start: "+pkgName))
	// Traverse blocks for phase-2
	if err := nci.translate(); err != nil {
		log(ftl(err))
		return err
	}
	log(hi("Phase-2"), "if err != nil check insertion", hi("end: "+pkgName))

	log("Translation", hi("end: "+pkgName))
	return nil
}
