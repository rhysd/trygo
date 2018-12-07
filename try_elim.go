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

type tempIDGen struct {
	id        uint
	generated map[*ast.Ident]struct{}
}

func (gen *tempIDGen) newTempID() *ast.Ident {
	ident := ast.NewIdent(fmt.Sprintf("_%d", gen.id))
	gen.generated[ident] = struct{}{}
	gen.id++
	return ident
}

func (gen *tempIDGen) isTempID(expr ast.Expr) bool {
	i, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	_, ok = gen.generated[i]
	return ok
}

func newTempIDGen() *tempIDGen {
	return &tempIDGen{
		id:        0,
		generated: map[*ast.Ident]struct{}{},
	}
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
	tempID     *tempIDGen
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

// Check given call expression is valid try() call and returns inner function call.
// The first return value is an inner call of try() call. The second return value is whether it's
// well-formed try() call or not.
// When inner it's not valid try() call, it returns false as the second return value.
// When the given call expression is actually not try() but it's still valid, it returns nil as
// the first return value though the second return value is true.
func (tce *tryCallElimination) checkTryCall(tryCall *ast.CallExpr) (*ast.CallExpr, bool) {
	name, ok := tryCall.Fun.(*ast.Ident)
	if !ok || name.Name != "try" {
		log("Skipped since callee is not 'try' function")
		return nil, true
	}

	if len(tryCall.Args) != 1 {
		tce.errfAt(tryCall, "try() should take 1 argument but %d arguments passed", len(tryCall.Args))
		return nil, false
	}

	inner, ok := tryCall.Args[0].(*ast.CallExpr)
	if !ok {
		tce.errfAt(tryCall, "try() call's argument must be function call but found %s", reflect.TypeOf(tryCall.Args[0]))
		return nil, false
	}

	if len(tce.funcs) == 0 {
		tce.errAt(tryCall, "try() function is used outside function")
		return nil, false
	}

	var funcTy *ast.FuncType
	switch f := tce.funcs.top().(type) {
	case *ast.FuncLit:
		funcTy = f.Type
	case *ast.FuncDecl:
		funcTy = f.Type
	}

	if funcTy.Results == nil || len(funcTy.Results.List) == 0 {
		tce.errAt(tryCall, "The function returns nothing. try() is not available")
		return nil, false
	}
	rets := funcTy.Results.List

	ty := rets[len(rets)-1].Type
	if ident, ok := ty.(*ast.Ident); !ok || ident.Name != "error" {
		tce.errfAt(tryCall, "The function does not return error as last return value. Last return type is %q", ty)
		return nil, false
	}

	log(hi("try() found"), "at", tce.logPos(tryCall), ':', inner.Fun)
	return inner, true
}

// checkMaybeTryCall checks given try() call and returns try() call and inner call (the argument of the try call)
// since try()'s argument must be function call. When it is not a try() call, it returns nil as first the
// return value. When it is an invalid try() call, it sets the error to err field and returns false
// as the third return value.
func (tce *tryCallElimination) checkMaybeTryCall(maybeCall ast.Expr) (tryCall *ast.CallExpr, innerCall *ast.CallExpr, ok bool) {
	outer, ok := maybeCall.(*ast.CallExpr)
	if !ok {
		log("Skipped since expression is not a call expression")
		return nil, nil, true
	}

	inner, ok := tce.checkTryCall(outer)
	if !ok || inner == nil {
		return nil, nil, ok
	}

	return outer, inner, true
}

func (tce *tryCallElimination) eliminateTryCall(kind transKind, node ast.Node, maybeTryCall ast.Expr) bool {
	tryCall, innerCall, ok := tce.checkMaybeTryCall(maybeTryCall)
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

	log(hi("Value spec translated"), "at", pos, "Added new translation point:", transKindValueSpec)
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

	if assign.Tok != token.DEFINE && assign.Tok != token.ASSIGN {
		tce.errfAt(assign, "try() can only be used with = and :=, but found %s. Note that compound assignments such as += are not supported", assign.Tok)
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

	log(hi("Assignment translated"), "at", hi(pos), "Added new translation point:", transKindAssign)
}

func (tce *tryCallElimination) visitToplevelExpr(stmt *ast.ExprStmt) {
	pos := tce.logPos(stmt)
	log("Toplevel call at", pos)

	if ok := tce.eliminateTryCall(transKindToplevelCall, stmt, stmt.X); ok {
		log(hi("Toplevel call translated"), "at", pos, "Added new translation point:", transKindToplevelCall)
		return
	}

	if tce.err == nil {
		// Recursively visit an expression in ExprStmt. This is necessary to find out non-translated
		// try() calls to make an error
		ast.Walk(tce, stmt.X)
	}
}

// Returns parent's current index and var ID
func (tce *tryCallElimination) pushBlock(node ast.Stmt) (int, *tempIDGen) {
	parent := tce.currentBlk
	tree := &blockTree{ast: node, parent: parent}
	if tree.isRoot() {
		log("New root block added")
		tce.roots = append(tce.roots, tree)
	} else {
		parent.children = append(parent.children, tree)
	}

	prevIdx, prevTempID := tce.blkIndex, tce.tempID

	tce.parentBlk = parent
	tce.currentBlk = tree
	tce.blkIndex = 0
	tce.tempID = newTempIDGen()

	return prevIdx, prevTempID
}

func (tce *tryCallElimination) popBlock(prevIdx int, prevTempID *tempIDGen) {
	tce.blkIndex = prevIdx
	tce.tempID = prevTempID
	tce.currentBlk = tce.parentBlk
	if tce.parentBlk != nil {
		tce.parentBlk = tce.parentBlk.parent
	}
}

func (tce *tryCallElimination) visitStmts(stmts []ast.Stmt) {
	for _, stmt := range stmts {
		if tce.err != nil {
			return
		}

		if e, ok := stmt.(*ast.ExprStmt); ok {
			tce.visitToplevelExpr(e)
		} else {
			// Recursively visit
			ast.Walk(tce, stmt)
		}

		// Note: Index of stmts is not available because inserting/removing statement requires to
		// adjust the current index.
		tce.blkIndex++
	}
}

func (tce *tryCallElimination) visitBlockNode(node ast.Stmt, list []ast.Stmt) {
	pos := tce.logPos(node)
	ty := reflect.TypeOf(node)
	log(hi("Block in ", ty, " start"), "at", pos)

	tce.parents = tce.parents.push(node)
	prevIdx, prevTempID := tce.pushBlock(node)
	tce.visitStmts(list)
	tce.popBlock(prevIdx, prevTempID)
	tce.parents = tce.parents.pop()

	log(hi("Block in ", ty, " end"), "at", pos)
}

// insertStmt inserts given statement *before* current index of current block
func (tce *tryCallElimination) insertStmt(stmt ast.Stmt) {
	tce.currentBlk.insertStmtAt(tce.blkIndex, stmt)
	// New statement was inserted. Adjust current index
	tce.blkIndex++
}

// makeIndirectExpr create an indirect expression with inserting an assignment statement.
// Before:
//   expr
// After:
//   $n := expr
//   $n
// When the expression is target try() call,
// func (tce *tryCallElimination) makeIndirectExpr(expr ast.Expr, tryCall *ast.CallExpr) (ast.Expr, bool) {
//
// }

func (tce *tryCallElimination) visitTryCallExpr(call *ast.CallExpr) ast.Visitor {
	innerCall, ok := tce.checkTryCall(call)
	if !ok {
		// Invalid try() call
		return nil
	}
	if innerCall == nil {
		// Not a try() call
		return tce
	}

	// var e ast.Expr
	// switch parent := tce.parents.top().(type) {

	// }

	// TODO: Implement translation for generic try() call expression
	tce.errAt(call, "try() call was not translated. Only try() calls at toplevel call expression, assignments (= or :=), value spec (var or const) are translated")
	return nil
}

func (tce *tryCallElimination) visitPre(node ast.Node) ast.Visitor {
	switch node := node.(type) {
	case *ast.CallExpr:
		return tce.visitTryCallExpr(node)
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
		log(hi("Start function:"), node.Name.Name)
	case *ast.FuncLit:
		tce.funcs = tce.funcs.push(node)
		log(hi("Start function literal"))
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
		log(hi("End function:"), node.Name.Name)
	case *ast.FuncLit:
		tce.funcs = tce.funcs.pop()
		log(hi("End function literal"))
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
