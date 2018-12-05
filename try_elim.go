package trygo

import (
	"fmt"
	"github.com/pkg/errors"
	"go/ast"
	"go/token"
	"reflect"
	"strings"
)

// Try call elimination.
//
// This pass eliminates all try() calls in source not to make type check fail.
// All try() call expressions are removed and '_' ignoring variable is inserted
// to declarations to receive error values.
//
// e.g.
//   x := try(f())  ->  x, _ := f()
//   x = try(f())   ->  x, _ = f()
//   try(f())       ->  f()

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

func (tce *tryCallElimination) nodePos(node ast.Node) token.Position {
	return tce.fileset.Position(node.Pos())
}

func (tce *tryCallElimination) logPos(node ast.Node) string {
	if !logEnabled {
		return ""
	}
	return relpath(tce.nodePos(node).String())
}

func (tce *tryCallElimination) errAt(node ast.Node, msg string) {
	tce.err = errors.Errorf("%s: %v: Error: %s", tce.nodePos(node), tce.pkg.Name, msg)
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
	log(hi("Eliminate try() call"), "for kind", kind, "at", tce.logPos(tryCall))

	// Squash try() call with inner call: try(f(...)) -> f(...)
	*tryCall = *innerCall

	p := &transPoint{
		kind:       kind,
		node:       node,
		blockIndex: tce.blkIndex,
		fun:        tce.funcs.top(),
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
	pos := tce.logPos(spec)
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
	spec.Names = append(spec.Names, newIdent("_", spec.Pos()))

	log("Value spec at", pos, "added new translation point")
}

func (tce *tryCallElimination) visitAssign(assign *ast.AssignStmt) {
	pos := tce.logPos(assign)
	log("Assignment at", pos)

	if len(assign.Rhs) != 1 {
		// In Go, multiple LHS expressions means they does not return multiple values
		// Note: Following is ill-formed:
		//   fromF := F(), try(funcOnlyReturnErr())
		log("Skipped due to multiple RHS values")
		return
	}

	switch tce.parents.top().(type) {
	case *ast.BlockStmt, *ast.CommClause, *ast.CaseClause:
		// ok, go ahead
	default:
		// This assignment is not at toplevel, for example, `if x := e; ...` or `for x := range e`...
		// Only toplevel assignments (= or :=) should be translated to avoid wrong if err != nil check insertion
		log("Skipped non-toplevel assignment at", pos)
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
	assign.Lhs = append(assign.Lhs, newIdent("_", assign.Pos()))

	log("Assignment at", pos, "added new translation point")
}

func (tce *tryCallElimination) visitToplevelExpr(stmt *ast.ExprStmt) {
	pos := tce.logPos(stmt)
	log("Assignment at", pos)

	if ok := tce.eliminateTryCall(transKindToplevelCall, stmt, stmt.X); !ok {
		// Error
		return
	}

	log("New translation point for toplevel try() call at", pos)
}

// Returns parent's current index
func (tce *tryCallElimination) pushBlock(node ast.Stmt) int {
	parent := tce.currentBlk
	tree := &blockTree{ast: node, parent: parent}
	if tree.isRoot() {
		log("New root block added")
		tce.roots = append(tce.roots, tree)
	} else {
		parent.children = append(parent.children, tree)
	}

	tce.parentBlk = parent
	tce.currentBlk = tree
	return tce.blkIndex
}

func (tce *tryCallElimination) popBlock(prevIdx int) {
	tce.blkIndex = prevIdx
	tce.currentBlk = tce.parentBlk
	if tce.parentBlk != nil {
		tce.parentBlk = tce.parentBlk.parent
	}
}

func (tce *tryCallElimination) visitStmts(stmts []ast.Stmt) {
	for i, stmt := range stmts {
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
}

func (tce *tryCallElimination) visitBlockNode(node ast.Stmt, list []ast.Stmt) {
	pos := tce.logPos(node)
	ty := reflect.TypeOf(node)
	log(hi("Block in ", ty, " start"), "at", pos)

	prevIdx := tce.pushBlock(node)
	tce.visitStmts(list)
	tce.popBlock(prevIdx)

	log(hi("Block in ", ty, " end"), "at", pos)
}

func (tce *tryCallElimination) visitPre(node ast.Node) ast.Visitor {
	switch node := node.(type) {
	case *ast.BlockStmt:
		tce.visitBlockNode(node, node.List)
		return nil // visitBlockNode() recursively calls ast.Walk() in itself
	case *ast.CaseClause:
		tce.visitBlockNode(node, node.Body)
		return nil // visitBlockNode() recursively calls ast.Walk() in itself
	case *ast.CommClause:
		tce.visitBlockNode(node, node.Body)
		return nil // visitBlockNode() recursively calls ast.Walk() in itself
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
