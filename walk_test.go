/*
 * Copyright 2018. bigpigeon. All rights reserved.
 * Use of this source code is governed by a MIT style
 * license that can be found in the LICENSE file.
 */

package main

import (
	"github.com/stretchr/testify/assert"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestWalkInit(t *testing.T) {
	src := standardSrc()
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, "src.go", src, 0|parser.ParseComments)
	assert.Nil(t, err)
	walk, err := NewWalker(fs, ".", []*ast.File{file}, true)
	if err != nil {
		panic(err)
	}
	cmap := ast.NewCommentMap(fs, file, file.Comments)

	ast.Inspect(file, func(node ast.Node) bool {
		switch x := node.(type) {
		case ast.Stmt:
			if cGroups, ok := cmap[x]; ok {

				lastCommends := cGroups[len(cGroups)-1].List
				lastCommand := lastCommends[len(lastCommends)-1]
				switch lastCommand.Text {
				case "// toy model":
					assign := x.(*ast.AssignStmt)
					node := assign.Rhs[0].(*ast.CallExpr).Fun
					name := walk.Info.Uses[node.(*ast.SelectorExpr).Sel].String()
					assert.Equal(t, walk.ToyModel.String(), name)
					t.Logf("obj name %s\n", name)
				case "// chain funcs":
					blockStmt := x.(*ast.BlockStmt)
					for _, stmt := range blockStmt.List {
						assign := stmt.(*ast.AssignStmt)
						node := assign.Rhs[0]
						if obj := walk.Info.Uses[node.(*ast.SelectorExpr).Sel]; obj != nil {
							t.Logf("obj name %s\n", obj.String())
							_, ok := walk.ToyChainMethod[obj.String()]
							assert.True(t, ok)
						}

					}
				case "// preload":
					assign := x.(*ast.AssignStmt)
					node := assign.Rhs[0]
					name := walk.Info.Uses[node.(*ast.SelectorExpr).Sel].String()
					assert.Equal(t, walk.ToyChainPreload.String(), name)
					t.Logf("obj name %s\n", name)
				case "// enter":
					assign := x.(*ast.AssignStmt)
					node := assign.Rhs[0]
					name := walk.Info.Uses[node.(*ast.SelectorExpr).Sel].String()
					assert.Equal(t, walk.ToyChainEnter.String(), name)
					t.Logf("obj name %s\n", name)
				case "// join":
					assign := x.(*ast.AssignStmt)
					node := assign.Rhs[0]
					name := walk.Info.Uses[node.(*ast.SelectorExpr).Sel].String()
					assert.Equal(t, walk.ToyChainJoin.String(), name)
					t.Logf("obj name %s\n", name)
				case "// swap":
					assign := x.(*ast.AssignStmt)
					node := assign.Rhs[0]
					name := walk.Info.Uses[node.(*ast.SelectorExpr).Sel].String()
					assert.Equal(t, walk.ToyChainSwap.String(), name)
					t.Logf("obj name %s\n", name)
				case "// offsetof":
					assign := x.(*ast.AssignStmt)
					node := assign.Rhs[0].(*ast.CallExpr).Fun
					name := walk.Info.Uses[node.(*ast.SelectorExpr).Sel].String()
					assert.Equal(t, walk.TypOffsetof.String(), name)
					t.Logf("obj name %s\n", name)
				}

			}
		}
		return true
	})
}

func TestWalk(t *testing.T) {
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, "testdata/struct_notmatch.go", nil, 0)
	assert.Nil(t, err)
	walk, err := NewWalker(fs, ".", []*ast.File{file}, true)
	assert.Nil(t, err)
	ast.Walk(walk, file)
	t.Logf("\n%s\n", walk.Report())
}
