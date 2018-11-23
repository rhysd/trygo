package trygo

import (
	"fmt"
	"github.com/pkg/errors"
	"go/ast"
	"go/token"
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

type funcTypeStack []*ast.FuncType

func (ts funcTypeStack) push(t *ast.FuncType) funcTypeStack {
	return append(ts, t)
}
func (ts funcTypeStack) pop() funcTypeStack {
	return ts[:len(ts)-1]
}
func (ts funcTypeStack) top() *ast.FuncType {
	return ts[len(ts)-1]
}

// Trans represents a state of translation
type Trans struct {
	Package    *ast.Package
	Files      *token.FileSet
	Err        error
	block      *ast.BlockStmt
	blockIndex int
	blockPos   string
	file       *ast.File
	varID      int
	parents    nodeStack
	funcs      funcTypeStack
}

func (trans *Trans) errAt(node ast.Node, msg string) {
	pos := trans.Files.Position(node.Pos())
	trans.Err = errors.Errorf("%s: %v: Error: %s", pos, trans.Package.Name, msg)
	log(trans.Err)
}

func (trans *Trans) errfAt(node ast.Node, format string, args ...interface{}) {
	trans.errAt(node, fmt.Sprintf(format, args...))
}

func (trans *Trans) genIdent() *ast.Ident {
	id := trans.varID
	trans.varID++
	return ast.NewIdent(fmt.Sprintf("_%d", id))
}

// insertStmts inserts given statements *before* current statement position
func (trans *Trans) insertStmts(stmts []ast.Stmt) {
	prev := trans.block.List
	l, r := prev[:trans.blockIndex], prev[trans.blockIndex:]
	ls := make([]ast.Stmt, 0, len(prev)+len(stmts))
	ls = append(ls, l...)
	ls = append(ls, stmts...)
	ls = append(ls, r...)
	trans.block.List = ls
	trans.blockIndex += len(stmts)
	log(hi(len(stmts)), "statements inserted to block at", trans.blockPos)
}

// insertIfErrChk generate error check if statement and insert it to current position.
// And returns identifiers of variables which bind return values of the given call.
//
//   Before:
//     foo()
//   After:
//     _0, _1, err := foo()
//     if err != nil {
//       return $retvals, err
//     }
func (trans *Trans) insertIfErrChk(call *ast.CallExpr, numRet int) []ast.Expr {
	inserted := make([]ast.Stmt, 0, 2)

	// Generate LHS of the assignment
	idents := make([]ast.Expr, 0, numRet+1) // +1 for err
	for i := 0; i < numRet; i++ {
		idents = append(idents, trans.genIdent())
	}
	errIdent := ast.NewIdent("err")
	idents = append(idents, errIdent)

	// Generate _0, _1, err := $call
	inserted = append(inserted, &ast.AssignStmt{
		Lhs: idents,
		Tok: token.DEFINE,
		Rhs: []ast.Expr{call},
	})
	log("Generate", hi("$ret, err :="), "assignment for", call.Fun)

	// Generate if err != nil { return err }
	inserted = append(inserted, &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  errIdent,
			Y:  ast.NewIdent("nil"),
			Op: token.NEQ,
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						// TODO: Calculate and add $retvals from visiting function declaration
						errIdent,
					},
				},
			},
		},
	})
	log("Generate", hi("if err != nil"), "check for", call.Fun)

	trans.insertStmts(inserted)

	return idents[:len(idents)-1] // Omit last 'err'
}

// checkTryCall checks given try() call and returns inner call (the argument of the try call) since
// try()'s argument must be function call. When it is not a try() call, it returns nil as first the
// return value. When it is an invalid try() call, it sets the error to Err field and returns false
// as the second return value.
func (trans *Trans) checkTryCall(call *ast.CallExpr) (*ast.CallExpr, bool) {
	name, ok := call.Fun.(*ast.Ident)
	if !ok {
		log("Skipped since callee was not var ref")
		return nil, true
	}
	if name.Name != "try" {
		log("Skipped since RHS is not calling 'try':", name.Name)
		return nil, true
	}

	if len(call.Args) != 1 {
		trans.errfAt(call, "try() takes 1 argument but %d found", len(call.Args))
		return nil, false
	}

	// TODO: Check callee returns an error as the last return value
	inner, ok := call.Args[0].(*ast.CallExpr)
	if !ok {
		trans.errfAt(call, "try() call's argument must be function call but found %s", reflect.TypeOf(call.Args[0]))
		return nil, false
	}

	if len(trans.funcs) == 0 {
		trans.errAt(call, "try() function is used outside function")
		return nil, false
	}

	rets := trans.funcs.top().Results.List
	if len(rets) == 0 {
		trans.errAt(call, "The function returns nothing. try() is not available")
		return nil, false
	}

	ty := rets[len(rets)-1].Type
	if ident, ok := ty.(*ast.Ident); !ok || ident.Name != "error" {
		trans.errfAt(call, "The function does not return error as last return value. Last return type is %q", ty)
		return nil, false
	}

	log(hi("try() found:"), inner.Fun)
	return inner, true
}

func (trans *Trans) transDefRHS(rhs []ast.Expr, numRet int) ([]ast.Expr, bool) {
	if len(rhs) != 1 {
		return nil, false
	}

	maybeTryCall, ok := rhs[0].(*ast.CallExpr)
	if !ok {
		log("Skipped since RHS is not a call expression")
		return nil, false
	}

	call, ok := trans.checkTryCall(maybeTryCall)
	if !ok {
		return nil, false
	}
	if call == nil {
		return nil, false
	}

	tmpIdents := []ast.Expr{}
	for _, ident := range trans.insertIfErrChk(call, numRet) {
		tmpIdents = append(tmpIdents, ident)
	}

	return tmpIdents, true
}

// visitSpec visits var $ident = ... or const $ident = ... for translation
//   Before:
//     var x = try(foo())
//   After:
//     $tmp, err := foo()
//     if err != nil {
//         return err
//     }
//     var x = $tmp
func (trans *Trans) visitSpec(spec *ast.ValueSpec) {
	pos := trans.Files.Position(spec.Pos())
	log("Value spec", pos)
	idents, ok := trans.transDefRHS(spec.Values, len(spec.Names))
	if !ok {
		return
	}
	log(hi("Value spec translated"), "Assignee:", hi(spec.Names), "Tmp:", idents, "at", pos)
	spec.Values = idents
}

func (trans *Trans) visitAssign(assign *ast.AssignStmt) {
	pos := trans.Files.Position(assign.Pos())
	log("Assignment block", pos)
	idents, ok := trans.transDefRHS(assign.Rhs, len(assign.Lhs))
	if !ok {
		return
	}
	log(hi("Assignment translated"), "Assignee:", hi(assign.Lhs), "Tmp:", idents, "at", pos)
	assign.Rhs = idents
}

func (trans *Trans) visitBlock(block *ast.BlockStmt) {
	pos := trans.Files.Position(block.Pos()).String()
	log("Block statement start", pos)
	b := trans.block          // push
	id := trans.varID         // push
	prevPos := trans.blockPos // push
	trans.block = block
	trans.varID = 0
	trans.blockIndex = 0
	trans.blockPos = pos
	list := block.List // This assignment is necessary since block.List is modified?
	for _, stmt := range list {
		ast.Walk(trans, stmt)
		trans.blockIndex++ // Cannot use index of this for loop since some statements may be inserted
	}
	trans.block = b          // pop
	trans.varID = id         // pop
	trans.blockPos = prevPos // pop
	log("Block statement end", pos)
}

func (trans *Trans) visitPre(node ast.Node) {
	switch node := node.(type) {
	case *ast.BlockStmt:
		trans.visitBlock(node)
	case *ast.ValueSpec:
		// var or const
		trans.visitSpec(node)
	case *ast.AssignStmt:
		// := or =
		trans.visitAssign(node)
	case *ast.FuncDecl:
		trans.funcs = trans.funcs.push(node.Type)
	case *ast.FuncLit:
		trans.funcs = trans.funcs.push(node.Type)
	case *ast.File:
		trans.file = node
	}
}

func (trans *Trans) visitPost(node ast.Node) {
	switch node.(type) {
	case *ast.FuncDecl:
		trans.funcs = trans.funcs.pop()
	case *ast.FuncLit:
		trans.funcs = trans.funcs.pop()
	}
}

func (trans *Trans) Visit(node ast.Node) ast.Visitor {
	if trans.Err != nil {
		return nil
	}

	if node == nil {
		n := trans.parents.top()
		trans.parents = trans.parents.pop()
		trans.visitPost(n)
		return nil
	}

	trans.visitPre(node)

	trans.parents = trans.parents.push(node)

	// When no error occurred, always visit children. Stopping visiting children collapses parents stack.
	// Note: It may be OK to return nil here. When return value is nil, we would also need to pop parents stack.
	return trans
}

// Translate is an entrypoint of translation. It translates TryGo code into Go code by modifying given
// AST directly and returns error if happens
func (trans *Trans) Translate() error {
	name := trans.Package.Name
	log("Translation", hi("start: "+name))
	ast.Walk(trans, trans.Package)
	log("Translation", hi("done: "+name))

	// Check parents stack did not collapsed
	if trans.Err == nil && len(trans.parents) != 0 {
		ss := make([]string, 0, len(trans.parents))
		for _, n := range trans.parents {
			ss = append(ss, reflect.TypeOf(n).String())
		}
		s := strings.Join(ss, " -> ")
		if s == "" {
			s = "(empty)"
		}
		panic(fmt.Sprintf("Parents stack collapsed: TOP -> %s -> BOTTOM", s))
	}

	return trans.Err
}

// NewTrans creates a new Trans instance with given package AST and tokens
func NewTrans(pkg *ast.Package, fs *token.FileSet) *Trans {
	return &Trans{
		Package:    pkg,
		Files:      fs,
		Err:        nil,
		block:      nil,
		blockIndex: 0,
		blockPos:   "(toplevel)",
		file:       nil,
		varID:      0,
		parents:    nil,
		funcs:      nil,
	}
}

// Translate translates TryGo code into Go code by modifying given AST directly
func Translate(pkg *ast.Package, fs *token.FileSet) error {
	return NewTrans(pkg, fs).Translate()
}
