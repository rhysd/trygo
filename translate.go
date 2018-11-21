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
	file       *ast.File
	varID      int
}

func (trans *Trans) errAt(node ast.Node, msg string) {
	pos := trans.Files.Position(node.Pos())
	trans.Err = errors.Errorf("%s: %v: Error: %s", pos, trans.Package.Name, msg)
}

func (trans *Trans) errfAt(node ast.Node, format string, args ...interface{}) {
	trans.errAt(node, fmt.Sprintf(format, args...))
}

func (trans *Trans) genIdent() *ast.Ident {
	id := trans.varID
	trans.varID++
	return ast.NewIdent(fmt.Sprintf("_%d", id))
}

// insertStmt inserts given statement *before* current statement position
func (trans *Trans) insertStmt(stmt ast.Stmt) {
	l, r := trans.block.List[:trans.blockIndex], trans.block.List[trans.blockIndex:]
	ls := make([]ast.Stmt, 0, len(trans.block.List)+1)
	ls = append(ls, l...)
	ls = append(ls, stmt)
	ls = append(ls, r...)
	trans.block.List = ls
	trans.blockIndex++
}

// insertStmts inserts given statements *before* current statement position
func (trans *Trans) insertStmts(stmts []ast.Stmt) {
	l, r := trans.block.List[:trans.blockIndex], trans.block.List[trans.blockIndex:]
	ls := make([]ast.Stmt, 0, len(trans.block.List)+len(stmts))
	ls = append(ls, l...)
	ls = append(ls, stmts...)
	ls = append(ls, r...)
	trans.block.List = ls
	trans.blockIndex += len(stmts)
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
func (trans *Trans) insertIfErrChk(call *ast.CallExpr, numRet int) []*ast.Ident {
	inserted := make([]ast.Stmt, 0, 2)

	// Generate _0, _1, err := $call
	names := make([]*ast.Ident, 0, numRet+1) // +1 for err
	for i := 0; i < numRet; i++ {
		names = append(names, trans.genIdent())
	}
	errIdent := ast.NewIdent("err")
	names = append(names, errIdent)
	def := &ast.ValueSpec{
		Names:  names,
		Values: []ast.Expr{call},
	}
	decl := &ast.GenDecl{
		Tok:   token.VAR,
		Specs: []ast.Spec{def},
	}
	inserted = append(inserted, &ast.DeclStmt{Decl: decl})

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

	trans.insertStmts(inserted)

	return names[:len(names)-1]
}

func (trans *Trans) checkTryCall(call *ast.CallExpr) (*ast.CallExpr, bool) {
	name, ok := call.Fun.(*ast.Ident)
	if !ok || name.Name != "try" {
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

	return inner, true
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
	if len(spec.Values) != 1 {
		return trans
	}

	maybeTryCall, ok := spec.Values[0].(*ast.CallExpr)
	if !ok {
		return trans
	}

	call, ok := trans.checkTryCall(maybeTryCall)
	if !ok {
		return nil
	}
	if call == nil {
		return trans
	}

	tmpIdents := []ast.Expr{}
	for _, ident := range trans.insertIfErrChk(call, len(spec.Names)) {
		tmpIdents = append(tmpIdents, ident)
	}

	spec.Values = []ast.Expr(tmpIdents)

	return trans
}

func (trans *Trans) visitBlock(block *ast.BlockStmt) ast.Visitor {
	b := trans.block  // push
	id := trans.varID // push
	trans.block = block
	trans.varID = 0
	list := block.List // This assignment is necessary since block.List is modified?
	for i, stmt := range list {
		trans.blockIndex = i
		ast.Walk(trans, stmt)
	}
	trans.block = b  // pop
	trans.varID = id // pop
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
	ast.Walk(trans, trans.Package)
	return trans.Err
}

// NewTrans creates a new Trans instance with given package AST and tokens
func NewTrans(pkg *ast.Package, fs *token.FileSet) *Trans {
	return &Trans{pkg, fs, nil, nil, 0, nil, 0}
}

// Translate translates TryGo code into Go code by modifying given AST directly
func Translate(pkg *ast.Package, fs *token.FileSet) error {
	return NewTrans(pkg, fs).Translate()
}
