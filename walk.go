/*
 * Copyright 2018. bigpigeon. All rights reserved.
 * Use of this source code is governed by a MIT style
 * license that can be found in the LICENSE file.
 */

package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"sort"
)

type ErrDifferentStruct struct {
	FileSet *token.FileSet
	Source  *types.Named
	Target  ast.Expr
}

func (e ErrDifferentStruct) Error() string {
	return fmt.Sprintf("%s type must same as %s", e.FileSet.Position(e.Target.Pos()), e.FileSet.Position(e.Source.Obj().Pos()))
}

type ErrInvalidField struct {
	FileSet *token.FileSet
	Source  *types.Named
	Expr    ast.Expr
}

func (e ErrInvalidField) Error() string {
	return fmt.Sprintf("%s field not found in %s", e.FileSet.Position(e.Expr.Pos()), e.FileSet.Position(e.Source.Obj().Pos()))
}

type ErrInvalidStructField struct {
	FileSet *token.FileSet
	Expr    ast.Expr
}

func (e ErrInvalidStructField) Error() string {
	return fmt.Sprintf("%s is not a struct field", e.FileSet.Position(e.Expr.Pos()))
}

type Walker struct {
	FS              *token.FileSet
	Toyorm          bool
	BrickIdentCache map[types.Object]TypesStructList
	BrickCallCache  map[*ast.CallExpr]TypesStructList
	Files           []*ast.File
	Info            *types.Info
	// Toy.Model method
	ToyModel *types.Func
	// all ToyBrick method those return type are itself
	ToyChainMethod map[string]struct{}
	// method with Preload/Join and Enter/Join
	ToyChainPreload *types.Func
	ToyChainEnter   *types.Func
	ToyChainJoin    *types.Func
	ToyChainSwap    *types.Func
	// unsafe.Offsetof func
	TypOffsetof *types.Builtin
	// type wtih toyorm.FieldSelection
	TypFieldSelection types.Type

	CheckedExpr map[ast.Expr]struct{}
	ErrorExpr   map[ast.Expr][]error

	Verbose bool
}

// copy not copy
func (w *Walker) copy() *Walker {
	newt := *w
	newt.BrickIdentCache = map[types.Object]TypesStructList{}
	for key := range w.BrickIdentCache {
		newt.BrickIdentCache[key] = w.BrickIdentCache[key]
	}
	newt.BrickCallCache = map[*ast.CallExpr]TypesStructList{}
	for key := range w.BrickCallCache {
		newt.BrickCallCache[key] = w.BrickCallCache[key]
	}
	return &newt
}

func (w *Walker) Report() string {
	// sort expr by position
	var exprs []ast.Expr
	for expr := range w.CheckedExpr {
		exprs = append(exprs, expr)
	}
	sort.Slice(exprs, func(i, j int) bool {
		return exprs[i].Pos() < exprs[j].Pos()
	})
	s := ""
	for _, e := range exprs {
		if w.Verbose {
			if len(w.ErrorExpr[e]) != 0 {
				s += fmt.Sprintf("%s has error:\n", w.FS.Position(e.Pos()))
			} else {
				s += fmt.Sprintf("%s ok\n", w.FS.Position(e.Pos()))
			}
		}
		for _, err := range w.ErrorExpr[e] {
			s += fmt.Sprintf("\t%s\n", err)
		}
	}
	return s
}

func (w *Walker) cacheToyorm(spec *ast.ImportSpec) {
	if spec.Path.Value == "\"github.com/bigpigeon/toyorm\"" {
		w.Toyorm = true
	}
}

func (w *Walker) getIdentMapWithBrickVar(spec *ast.ValueSpec) {
	// ident only map one result call expr or other ident
	identMap := map[*ast.Ident]ast.Expr{}
	// j use to index rhs
	j := 0
	for i := range spec.Values {
		switch x := spec.Values[i].(type) {
		case *ast.CallExpr:
			if sign, ok := w.Info.Types[x.Fun].Type.(*types.Signature); ok {
				if sign.Results().Len() == 1 {
					identMap[spec.Names[i]] = x
				}
				j += sign.Results().Len()
			}
		default:
			identMap[spec.Names[i]] = x
			j++
		}
	}
	w.cacheBrickIdent(identMap)
}

func (w *Walker) getIdentMapWIthBrickAssign(stmt *ast.AssignStmt) {
	// ident only map one result call expr or other ident
	identMap := map[*ast.Ident]ast.Expr{}
	// j use to index rhs
	j := 0
	for i := range stmt.Rhs {
		switch x := stmt.Rhs[i].(type) {
		case *ast.CallExpr:
			if sign, ok := w.Info.Types[x.Fun].Type.(*types.Signature); ok {
				if sign.Results().Len() == 1 {
					if lhIdent := getIdent(stmt.Lhs[i]); lhIdent != nil {
						identMap[lhIdent] = x
					}
				}
				j += sign.Results().Len()
			}
		default:
			if lhIdent := getIdent(stmt.Lhs[i]); lhIdent != nil {
				identMap[lhIdent] = x
			}
			j++
		}
	}
	w.cacheBrickIdent(identMap)
}

func (w *Walker) cacheBrickIdent(identMap map[*ast.Ident]ast.Expr) {
	toyBrickType := w.ToyModel.Type().(*types.Signature).Results().At(0).Type()
	for lhIdent, expr := range identMap {
		// token is =, obj in w.Info.User, otherwise in w.Info.Defs
		var lhObj types.Object
		if obj, ok := w.Info.Uses[lhIdent]; ok && obj.Type().String() == toyBrickType.String() {
			lhObj = obj
		} else if obj, ok := w.Info.Defs[lhIdent]; ok && obj != nil && obj.Type().String() == toyBrickType.String() {
			lhObj = obj
		}
		if lhObj != nil {
			// if rhs is ToyBrick Chain function
			if call, ok := expr.(*ast.CallExpr); ok && w.Info.Types[call].Type.String() == toyBrickType.String() {
				w.checkCallExpr(call)
				if ctx, ok := w.BrickCallCache[call]; ok {
					w.BrickIdentCache[lhObj] = ctx
				}
			} else if ident := getIdent(expr); ident != nil {
				// if rhs is other *ToyBrick
				if ctx, ok := w.BrickIdentCache[w.Info.Uses[ident]]; ok {
					w.BrickIdentCache[lhObj] = ctx
				}
			}
		}
	}
}

func (w *Walker) IsBrickChain(obj types.Object) bool {
	if _, ok := w.ToyChainMethod[obj.String()]; ok {
		return true
	}
	return false
}

// check all ToyBrick chain syntax
// e.g
// brick := toy.Model(Product{}).Preload(Offsetof(Product{}.Detail))  ............ ok, Preload struct same as Model struct
// brick := toy.Model(Product{}).Preload(Offsetof(User{}.Detail)) ................ error, Preload struct not match Model struct
func (w *Walker) ArgsCheck(mType *types.Named, args ...ast.Expr) {
	for _, expr := range args {
		switch x := expr.(type) {
		case *ast.CompositeLit:
			if _, ok := x.Type.(*ast.MapType); ok {
				w.ArgsCheck(mType, x.Elts...)
			}
		case *ast.KeyValueExpr:
			w.ArgsCheck(mType, x.Key)
		case *ast.BasicLit:
			if x.Kind == token.STRING {
				w.CheckedExpr[x] = struct{}{}
				fieldMap, err := getStructFieldMap(mType.Underlying().(*types.Struct), w.FS)
				if err != nil {
					w.ErrorExpr[x] = append(w.ErrorExpr[x], err)
					break
				}
				var name string
				err = json.Unmarshal([]byte(x.Value), &name)
				// error must be nil, panic for debug
				if err != nil {
					panic(err)
				}
				if _, ok := fieldMap[name]; ok == false {
					w.ErrorExpr[x] = append(w.ErrorExpr[x], ErrInvalidField{w.FS, mType, expr})
				}
			}
		case *ast.CallExpr:
			var obj types.Object
			var ok bool
			switch y := x.Fun.(type) {
			case *ast.Ident:
				obj, ok = w.Info.Uses[y]
			case *ast.SelectorExpr:
				obj, ok = w.Info.Uses[y.Sel]
			}

			if ok {
				if obj.String() == w.TypOffsetof.String() {
					w.CheckedExpr[x] = struct{}{}
					arg := x.Args[0]
					cType := getTypesStruct(w.Info.Types[arg.(*ast.SelectorExpr).X].Type)
					if cType != mType {
						w.ErrorExpr[x] = append(w.ErrorExpr[x], ErrDifferentStruct{w.FS, mType, arg})
					}
				}
			}
		}
	}
}

// check field must be struct
func (w *Walker) checkStructField(field ast.Expr, current *types.Struct) *types.Named {
	switch x := field.(type) {
	case *ast.CallExpr:
		var obj types.Object
		var ok bool
		switch y := x.Fun.(type) {
		case *ast.SelectorExpr:
			obj, ok = w.Info.Uses[y.Sel]
		case *ast.Ident:
			obj, ok = w.Info.Uses[y]
		}
		if ok && w.TypOffsetof.String() == obj.String() {
			w.CheckedExpr[x] = struct{}{}
			arg := x.Args[0].(*ast.SelectorExpr)
			if structType := getTypesStruct(w.Info.Selections[arg].Type()); structType != nil {
				return structType
			}
			w.ErrorExpr[x] = append(w.ErrorExpr[x], ErrInvalidStructField{w.FS, arg.Sel})
		}

	case *ast.BasicLit:
		if x.Kind == token.STRING {
			w.CheckedExpr[x] = struct{}{}
			fieldMap, err := getStructFieldMap(current, w.FS)
			if err != nil {
				w.ErrorExpr[x] = append(w.ErrorExpr[x], err)
				return nil
			}
			var name string
			err = json.Unmarshal([]byte(x.Value), &name)
			// error must be nil, panic for debug
			if err != nil {
				panic(err)
			}
			if field, ok := fieldMap[name]; ok {
				if structType := getTypesStruct(field.Type()); structType != nil {
					return structType
				}
			}
			w.ErrorExpr[x] = append(w.ErrorExpr[x], ErrInvalidStructField{w.FS, x})
		}
	}
	return nil
}

// get all toyorm.FieldSelection args in function
func (w *Walker) getFieldSelection(call *ast.CallExpr) []ast.Expr {
	var args []ast.Expr
	callTyp := w.Info.Types[call.Fun].Type.(*types.Signature)

	if callTyp.Variadic() {
		for i := 0; i < callTyp.Params().Len()-1; i++ {
			if callTyp.Params().At(i).Type().String() == w.TypFieldSelection.String() {
				args = append(args, call.Args[i])
			}
		}
		// in variadic arg must more than one
		lastArg := callTyp.Params().At(callTyp.Params().Len() - 1)
		// variadic arg must be slice type
		if lastArg.Type().(*types.Slice).Elem().String() == w.TypFieldSelection.String() {
			for i := callTyp.Params().Len() - 1; i < len(call.Args); i++ {
				args = append(args, call.Args[i])
			}
		}
	} else {
		for i := 0; i < callTyp.Params().Len(); i++ {
			if callTyp.Params().At(i).Type().String() == w.TypFieldSelection.String() {
				args = append(args, call.Args[i])
			}
		}
	}

	return args
}

// to check brick chain syntax
// output error when chain model different source model
// error e.g brick.Model(Product{}).OrderBy(Offsetof(User{}.Data))
func (w *Walker) checkCallExpr(call *ast.CallExpr) TypesStructList {
	var ctx TypesStructList
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		// prevent duplicate check
		if checkedList, ok := w.BrickCallCache[call]; ok {
			return checkedList
		}

		methodObj := w.Info.Uses[sel.Sel]
		// get previous ctx

		// TODO for the declarations
		// method := brick.Limit
		// method(2)

		// for the declarations
		// toy := ToyOpen("sqlite3", "")
		// brick = toy.Model(&Product{}).Debug.Where()...
		if selCall, ok := sel.X.(*ast.CallExpr); ok {
			ctx = w.checkCallExpr(selCall)
		} else if selIdent := getIdent(sel.X); selIdent != nil {
			ctx = w.BrickIdentCache[w.Info.Uses[selIdent]]
		}

		if w.ToyModel.String() == methodObj.String() {
			arg := call.Args[0]
			if _type, ok := w.Info.Types[arg]; ok {
				sType := getTypesStruct(_type.Type)
				if sType == nil {
					panic("args error")
				}
				ctx = append(ctx.Copy(), sType)
			}
		} else if len(ctx) > 0 {
			if w.IsBrickChain(methodObj) {
				w.ArgsCheck(ctx[len(ctx)-1], w.getFieldSelection(call)...)
			} else if w.ToyChainPreload.String() == methodObj.String() || w.ToyChainJoin.String() == methodObj.String() {
				w.ArgsCheck(ctx[len(ctx)-1], w.getFieldSelection(call)...)
				// check Preload field type
				if fieldStruct := w.checkStructField(call.Args[0], ctx[len(ctx)-1].Underlying().(*types.Struct)); fieldStruct != nil {
					ctx = append(ctx.Copy(), fieldStruct)
				} else {
					ctx = nil
				}
			} else if len(ctx) > 1 {
				if w.ToyChainEnter.String() == methodObj.String() || w.ToyChainSwap.String() == methodObj.String() {
					// enter and swap haven't args
					ctx = ctx[:len(ctx)-1]
				}
			}
		}

		// this call was checked
		w.BrickCallCache[call] = ctx
	}
	return ctx
}

func (w *Walker) Init() error {
	src := standardSrc()
	pFile, err := parser.ParseFile(w.FS, "standard.go", src, 0|parser.ParseComments)
	if err != nil {
		return err
	}
	config := types.Config{Importer: importer.For("source", nil), FakeImportC: true}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Uses:       make(map[*ast.Ident]types.Object),
	}
	_, err = config.Check("", w.FS, []*ast.File{pFile}, info)
	if err != nil {
		return err
	}
	cmap := ast.NewCommentMap(w.FS, pFile, pFile.Comments)

	ast.Inspect(pFile, func(node ast.Node) bool {
		switch x := node.(type) {
		case ast.Stmt:
			if cGroups, ok := cmap[x]; ok {
				lastCommends := cGroups[len(cGroups)-1].List
				lastCommand := lastCommends[len(lastCommends)-1]
				switch lastCommand.Text {
				case "// toy model":
					assign := x.(*ast.AssignStmt)
					node := assign.Rhs[0].(*ast.CallExpr).Fun
					w.ToyModel = info.Uses[node.(*ast.SelectorExpr).Sel].(*types.Func)
				case "// chain funcs":
					blockStmt := x.(*ast.BlockStmt)
					for _, stmt := range blockStmt.List {
						assign := stmt.(*ast.AssignStmt)
						node := assign.Rhs[0]
						if obj, ok := info.Uses[node.(*ast.SelectorExpr).Sel]; ok {
							w.ToyChainMethod[obj.String()] = struct{}{}
						}
					}
				case "// preload":
					assign := x.(*ast.AssignStmt)
					node := assign.Rhs[0]
					w.ToyChainPreload = info.Uses[node.(*ast.SelectorExpr).Sel].(*types.Func)
					// add field selection:
					w.TypFieldSelection = w.ToyChainPreload.Type().(*types.Signature).Params().At(0).Type()
				case "// enter":
					assign := x.(*ast.AssignStmt)
					node := assign.Rhs[0]
					w.ToyChainEnter = info.Uses[node.(*ast.SelectorExpr).Sel].(*types.Func)
				case "// join":
					assign := x.(*ast.AssignStmt)
					node := assign.Rhs[0]
					w.ToyChainJoin = info.Uses[node.(*ast.SelectorExpr).Sel].(*types.Func)
				case "// swap":
					assign := x.(*ast.AssignStmt)
					node := assign.Rhs[0]
					w.ToyChainSwap = info.Uses[node.(*ast.SelectorExpr).Sel].(*types.Func)
				case "// offsetof":
					assign := x.(*ast.AssignStmt)
					node := assign.Rhs[0].(*ast.CallExpr).Fun
					w.TypOffsetof = info.Uses[node.(*ast.SelectorExpr).Sel].(*types.Builtin)
				}
			}
		}
		return true
	})
	return nil
}

func NewWalker(fileSet *token.FileSet, path string, files []*ast.File, verbose bool) (*Walker, error) {
	walker := &Walker{
		FS:             fileSet,
		Files:          files,
		ToyChainMethod: map[string]struct{}{},
		Info: &types.Info{
			Uses:       map[*ast.Ident]types.Object{},
			Types:      map[ast.Expr]types.TypeAndValue{},
			Selections: map[*ast.SelectorExpr]*types.Selection{},
			Scopes:     map[ast.Node]*types.Scope{},
			Defs:       make(map[*ast.Ident]types.Object),
		},
		CheckedExpr: map[ast.Expr]struct{}{},
		ErrorExpr:   map[ast.Expr][]error{},
		Verbose:     verbose,
	}
	config := types.Config{Importer: importer.For("source", nil), FakeImportC: true}
	_, err := config.Check(path, walker.FS, walker.Files, walker.Info)
	if err != nil {
		return nil, err
	}
	if err := walker.Init(); err != nil {
		return nil, err
	}
	return walker, nil
}

func (w *Walker) Visit(node ast.Node) ast.Visitor {
	switch x := node.(type) {
	case *ast.ImportSpec:
		w.cacheToyorm(x)
	//if x.Tok != token.IMPORT {
	//	return nil
	//}
	case *ast.ValueSpec:
		w.getIdentMapWithBrickVar(x)
	case *ast.AssignStmt:
		w.getIdentMapWIthBrickAssign(x)
	case *ast.CallExpr:
		w.checkCallExpr(x)
	case *ast.BlockStmt:
		return w.copy()
	}
	return w
}
