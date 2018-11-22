package trygo

import (
	"fmt"
	"github.com/pkg/errors"
	"go/ast"
	"go/token"
	"reflect"
)

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
			Op: token.EQL,
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

func (trans *Trans) checkTryCall(call *ast.CallExpr) (*ast.CallExpr, bool) {
	name, ok := call.Fun.(*ast.Ident)
	if !ok || name.Name != "try" {
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

	log(hi("try() found:"), inner.Fun)
	return inner, true
}

func (trans *Trans) transDefRHS(rhs []ast.Expr, numRet int) (ast.Visitor, []ast.Expr, bool) {
	if len(rhs) != 1 {
		return trans, nil, false
	}

	maybeTryCall, ok := rhs[0].(*ast.CallExpr)
	if !ok {
		log("Skipped since RHS is not a call expression")
		return trans, nil, false
	}

	call, ok := trans.checkTryCall(maybeTryCall)
	if !ok {
		return nil, nil, false
	}
	if call == nil {
		return trans, nil, false
	}

	tmpIdents := []ast.Expr{}
	for _, ident := range trans.insertIfErrChk(call, numRet) {
		tmpIdents = append(tmpIdents, ident)
	}

	return trans, tmpIdents, true
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
func (trans *Trans) visitSpec(spec *ast.ValueSpec) ast.Visitor {
	pos := trans.Files.Position(spec.Pos())
	log("Value spec", pos)
	vis, idents, ok := trans.transDefRHS(spec.Values, len(spec.Names))
	if !ok {
		return vis
	}
	log(hi("Value spec translated"), "with idents", idents, "at", pos)
	spec.Values = idents
	return trans
}

func (trans *Trans) visitAssign(assign *ast.AssignStmt) ast.Visitor {
	pos := trans.Files.Position(assign.Pos())
	log("Assignment block", pos)
	vis, idents, ok := trans.transDefRHS(assign.Rhs, len(assign.Lhs))
	if !ok {
		return vis
	}
	log(hi("Assignment translated"), "with idents", idents, "at", pos)
	assign.Rhs = idents
	return trans
}

func (trans *Trans) visitBlock(block *ast.BlockStmt) ast.Visitor {
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
	return nil
}

func (trans *Trans) Visit(node ast.Node) ast.Visitor {
	if trans.Err != nil {
		return nil
	}
	switch node := node.(type) {
	case *ast.BlockStmt:
		return trans.visitBlock(node)
	case *ast.ValueSpec:
		// var or const
		return trans.visitSpec(node)
	case *ast.AssignStmt:
		// := or =
		return trans.visitAssign(node)
	case *ast.File:
		trans.file = node
		return trans
	default:
		return trans
	}
}

// Translate is an entrypoint of translation. It translates TryGo code into Go code by modifying given
// AST directly and returns error if happens
func (trans *Trans) Translate() error {
	n := hi(trans.Package.Name)
	log("Translation start:", n)
	ast.Walk(trans, trans.Package)
	log("Translation done:", n)
	return trans.Err
}

// NewTrans creates a new Trans instance with given package AST and tokens
func NewTrans(pkg *ast.Package, fs *token.FileSet) *Trans {
	return &Trans{pkg, fs, nil, nil, 0, "(toplevel)", nil, 0}
}

// Translate translates TryGo code into Go code by modifying given AST directly
func Translate(pkg *ast.Package, fs *token.FileSet) error {
	return NewTrans(pkg, fs).Translate()
}
