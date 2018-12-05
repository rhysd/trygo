package trygo

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
)

// Nil check insertion.
// It completes nil checks of try() calls by inserting AST nodes into statement blocks.
//
//   - Insert `if _err$n != nil { return $zerovals, _err$n }`.
//   - Replace '_' ignoring variables inserted by try() call elimination with _err$n variables.
//   - Inserts `var _err$n error` for assignments.
//   - On toplevel try(f()) call expression statement, the function call is replaced with
//     `if $ignores, err := f(); err != nil { ... }`.

func newIdent(name string, pos token.Pos) *ast.Ident {
	i := ast.NewIdent(name)
	i.NamePos = pos
	return i
}

type nilCheckInsertion struct {
	pkg      *ast.Package
	fileset  *token.FileSet
	roots    []*blockTree
	blk      *blockTree
	offset   int
	varID    int
	typeInfo *types.Info
	pkgTypes *types.Package
}

func (nci *nilCheckInsertion) nodePos(node ast.Node) token.Position {
	return nci.fileset.Position(node.Pos())
}

func (nci *nilCheckInsertion) logPos(node ast.Node) string {
	if !logEnabled {
		return ""
	}
	return relpath(nci.nodePos(node).String())
}

func (nci *nilCheckInsertion) genIdent(pos token.Pos) *ast.Ident {
	id := nci.varID
	nci.varID++
	return newIdent(fmt.Sprintf("_%d", id), pos)
}

func (nci *nilCheckInsertion) genErrIdent(pos token.Pos) *ast.Ident {
	i := newIdent(fmt.Sprintf("_err%d", nci.varID), pos)
	nci.varID++
	return i
}

func (nci *nilCheckInsertion) typeInfoFor(node ast.Expr) types.Type {
	t, ok := nci.typeInfo.Types[node]
	if !ok {
		panic("Type information is not collected for AST node at " + nci.nodePos(node).String())
	}
	return t.Type
}

func (nci *nilCheckInsertion) funcTypeOf(node ast.Node) (*types.Signature, *ast.FuncType) {
	if decl, ok := node.(*ast.FuncDecl); ok {
		obj, ok := nci.typeInfo.Defs[decl.Name]
		if !ok {
			// This case never occur in normal cases since type check passed. There must not be
			// unresolved identifiers whose types are unknown.
			panic(fmt.Sprintf("Type check was OK but type cannot be resolved for function '%s' at %s", decl.Name.Name, nci.nodePos(decl)))
		}
		ty := obj.Type().(*types.Signature)
		log("Function type of func", decl.Name.Name, "->", ty)
		return ty, decl.Type
	}

	lit := node.(*ast.FuncLit)
	ty := nci.typeInfoFor(lit).(*types.Signature)
	log("Function type of func literal at", nci.logPos(lit), "->", ty)
	return ty, lit.Type
}

// insertStmts inserts given statements *before* given index position of current block. If previous
// translation exists in the same block and some statements were already inserted, the offset is
// automatically adjusted.
func (nci *nilCheckInsertion) insertStmtAt(idx int, stmt ast.Stmt) {
	logf("Insert %T statements to block at %s", stmt, nci.logPos(nci.blk.ast))
	prev := nci.blk.stmts()
	idx += nci.offset
	l, r := prev[:idx], prev[idx:]
	ls := make([]ast.Stmt, 0, len(prev)+1)
	ls = append(ls, l...)
	ls = append(ls, stmt)
	ls = append(ls, r...)
	nci.blk.setStmts(ls)
	nci.offset++
}

func (nci *nilCheckInsertion) removeStmtAt(idx int) {
	prev := nci.blk.stmts()
	idx += nci.offset
	l, r := prev[:idx], prev[idx+1:]
	nci.blk.setStmts(append(l, r...))
	nci.offset--
	log(hi(idx+1, "th statement was removed from block at", nci.logPos(nci.blk.ast)))
}

func (nci *nilCheckInsertion) zeroValueOf(ty types.Type, typeNode ast.Expr, pos token.Pos) (expr ast.Expr) {
	tyStr := ty.String()
	log("Zero value will be calculated for", hi(tyStr))
	switch ty := ty.(type) {
	case *types.Basic:
		switch ty.Kind() {
		case types.Bool, types.UntypedBool, types.UntypedInt:
			expr = newIdent("false", pos)
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
			types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64,
			types.Uintptr:
			expr = &ast.BasicLit{
				Kind:     token.INT,
				Value:    "0",
				ValuePos: pos,
			}
		case types.Float32, types.Float64, types.UntypedFloat:
			expr = &ast.BasicLit{
				Kind:     token.FLOAT,
				Value:    "0.0",
				ValuePos: pos,
			}
		case types.Complex64, types.Complex128, types.UntypedComplex:
			expr = &ast.BasicLit{
				Kind:     token.IMAG,
				Value:    "0i",
				ValuePos: pos,
			}
		case types.String, types.UntypedString:
			expr = &ast.BasicLit{
				Kind:     token.STRING,
				Value:    `""`,
				ValuePos: pos,
			}
		case types.UnsafePointer, types.UntypedNil:
			expr = newIdent("nil", pos)
		case types.UntypedRune:
			expr = &ast.BasicLit{
				Kind:     token.CHAR,
				Value:    `'\0'`,
				ValuePos: pos,
			}
		default:
			panic("Unreachable")
		}
	case *types.Slice, *types.Pointer, *types.Signature, *types.Interface, *types.Map, *types.Chan:
		expr = newIdent("nil", pos)
	case *types.Struct, *types.Array:
		// To create CompositeLit for zero value of immediate struct, we reuse the AST node from return type of
		// function declaration because reconstruct immediate struct type AST node from *types.Struct needs bunch
		// of code for constructing ast.Expr from types.Type generally.
		// Note that position of AST node is not correct.
		expr = &ast.CompositeLit{Type: typeNode}
		log("AST type node at", nci.logPos(typeNode), "is reused to generate zero value of", reflect.TypeOf(typeNode))
	case *types.Named:
		u := ty.Underlying()
		if _, ok := u.(*types.Struct); ok {
			// When the underlying type is struct, CompositeLit should be used for zero value. To create it, we reuse
			// the AST node from return type of function declaration because it may contain package name like pkg.S.
			// There is no API to get package(pkg) and name(S) separately from types.Named. We need to parse string
			// representation. Reusing the AST node is better than parsing.
			// Note that position of AST node is not correct.
			expr = &ast.CompositeLit{Type: typeNode}
			log("AST type node at", nci.logPos(typeNode), "is reused to generate zero value of *types.Named")
			break
		}
		expr = nci.zeroValueOf(u, typeNode, pos)
	case *types.Tuple:
		panic("Cannot obtain zero value of tuple: " + tyStr)
	default:
		panic("Cannot obtain zero value of tuple: " + tyStr + ": " + reflect.TypeOf(ty).String())
	}

	log("Zero value:", hi(tyStr), "->", hi(reflect.TypeOf(expr)))
	return
}

func (nci *nilCheckInsertion) insertIfNilChkStmtAfter(index int, errIdent *ast.Ident, init ast.Stmt, fun ast.Node) {
	funcTy, funcTyNode := nci.funcTypeOf(fun)
	pos := errIdent.NamePos
	rets := funcTy.Results()
	retLen := rets.Len()
	retVals := make([]ast.Expr, 0, retLen)
	for i := 0; i < retLen-1; i++ { // -1 since last type is 'error'
		ret := rets.At(i).Type()
		node := funcTyNode.Results.List[i].Type
		retVals = append(retVals, nci.zeroValueOf(ret, node, pos))
	}
	retVals = append(retVals, errIdent)

	stmt := &ast.IfStmt{
		If:   pos,
		Init: init,
		Cond: &ast.BinaryExpr{
			X:     errIdent,
			Y:     newIdent("nil", pos),
			Op:    token.NEQ,
			OpPos: pos,
		},
		Body: &ast.BlockStmt{
			Lbrace: pos,
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: retVals,
					Return:  pos,
				},
			},
		},
	}

	nci.insertStmtAt(index+1, stmt)
	log("Inserted `if` statement for nil check at index", index+1, "of block at", nci.logPos(nci.blk.ast))
}

func (nci *nilCheckInsertion) transValueSpec(node *ast.ValueSpec, trans *transPoint) {
	// From:
	//   var $retvals, _ = f(...)
	// To:
	//   var $retvals, err = f(...)
	//   if err != nil {
	//     return $zerovals, err
	//   }
	errIdent := nci.genErrIdent(node.Pos())
	log(hi("Start value spec (var =)"), "translation", errIdent.Name)
	node.Names[len(node.Names)-1] = errIdent
	nci.insertIfNilChkStmtAfter(trans.blockIndex, errIdent, nil, trans.fun)
	log(hi("End value spec (var =)"), "translation", errIdent.Name)
	return
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
		errIdent := nci.genErrIdent(node.Pos())
		log(hi("Start define statement(:=)"), "translation", errIdent.Name)
		node.Lhs[len(node.Lhs)-1] = errIdent
		nci.insertIfNilChkStmtAfter(trans.blockIndex, errIdent, nil, trans.fun)
		log(hi("End define statement(:=)"), "translation", errIdent.Name)
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
	pos := node.Pos()
	errIdent := nci.genErrIdent(pos)
	log(hi("Start assign statement(=)"), "translation", errIdent.Name)
	decl := &ast.DeclStmt{
		Decl: &ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{
				&ast.ValueSpec{
					Names: []*ast.Ident{
						errIdent,
					},
					Type: newIdent("error", pos),
				},
			},
			TokPos: pos,
		},
	}
	// Insert `var _err$n error`
	nci.insertStmtAt(trans.blockIndex, decl)

	node.Lhs[len(node.Lhs)-1] = errIdent
	nci.insertIfNilChkStmtAfter(trans.blockIndex, errIdent, nil, trans.fun)
	log(hi("End assign statement(=)"), "translation", errIdent.Name)
}

func (nci *nilCheckInsertion) transToplevelExpr(trans *transPoint) {
	// From:
	//   f(...)
	// To:
	//   if $ignores, err := f(...); err != nil {
	//     return $zerovals, err
	//   }
	log(hi("Start toplevel try()"), "translation")

	// Remove the *ast.ExprStmt at first
	nci.removeStmtAt(trans.blockIndex)

	// Get the returned type of function call in try() invocation
	ty := nci.typeInfoFor(trans.call)

	// numIgnores is a number of '_' in LHS of if _, ..., err := ...
	numIgnores := 0
	if tpl, ok := ty.(*types.Tuple); ok {
		numIgnores = tpl.Len() - 1 // - 1 means omitting last 'error' type
	}

	log("Insert `if $ignores, err := ...; err != nil` check for", trans.kind, "with", numIgnores, "'_' var at", nci.logPos(trans.call))

	pos := trans.pos
	lhs := make([]ast.Expr, 0, numIgnores+1) // + 1 means the last 'error' variable
	for i := 0; i < numIgnores; i++ {
		lhs = append(lhs, newIdent("_", pos))
	}
	errIdent := newIdent("err", pos)
	lhs = append(lhs, errIdent)

	// Create err := ...
	assign := &ast.AssignStmt{
		Lhs:    lhs,
		Tok:    token.DEFINE,
		TokPos: pos,
		Rhs:    []ast.Expr{trans.call},
	}

	// Insert if err := ...; err != nil { ... }
	nci.insertIfNilChkStmtAfter(trans.blockIndex, errIdent, assign, trans.fun)

	log(hi("End toplevel try()"), "translation")
}

func (nci *nilCheckInsertion) insertNilCheck(trans *transPoint) {
	log(hi("Insert if err != nil check for "+trans.kind.String()), "at", nci.logPos(trans.node))

	switch trans.kind {
	case transKindValueSpec:
		nci.transValueSpec(trans.node.(*ast.ValueSpec), trans)
	case transKindAssign:
		nci.transAssign(trans.node.(*ast.AssignStmt), trans)
	case transKindToplevelCall:
		nci.transToplevelExpr(trans)
	case transKindExpr:
		panic("TODO: Translate non-toplevel try() call expressions")
	default:
		panic("Unreachable")
	}
}

func (nci *nilCheckInsertion) block(b *blockTree) {
	nci.blk = b
	nci.offset = 0
	nci.varID = 0

	pos := nci.logPos(b.ast)
	log("Start nil check insertion for block at", pos)
	for _, trans := range b.transPoints {
		nci.insertNilCheck(trans)
	}
	log("End nil check insertion for block at", pos)

	log("Recursively insert nil check to", hi(len(b.children)), "children in block at", pos)
	for _, child := range b.children {
		nci.block(child)
	}
}

func (nci *nilCheckInsertion) translate() {
	for _, root := range nci.roots {
		nci.block(root)
	}
}
