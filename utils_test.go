/*
 * Copyright 2018. bigpigeon. All rights reserved.
 * Use of this source code is governed by a MIT style
 * license that can be found in the LICENSE file.
 */

package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestTypesStruct(t *testing.T) {
	src := `
package p

import "unsafe"

type Product struct{
	ID int
	Name string
}

var product Product

func main(){
	var p1 Product
	p2 := p1
	_ = unsafe.Offsetof(p2.ID)
}

`
	fs := token.NewFileSet()
	parserFile, err := parser.ParseFile(fs, "src.go", src, 0)
	if err != nil {
		panic(err)
	}
	config := types.Config{Importer: importer.For("source", nil), FakeImportC: true}
	info := &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
		Scopes:     map[ast.Node]*types.Scope{},
		Defs:       make(map[*ast.Ident]types.Object),
	}
	//ast.Print(fs, parserFile)
	_, err = config.Check("./", fs, []*ast.File{parserFile}, info)
	if err != nil {
		t.Fatalf("checking package: %s", err)
	}
	pNum := 0
	ast.Inspect(parserFile, func(node ast.Node) bool {
		switch x := node.(type) {
		case ast.Expr:
			t := info.Types[x]
			if sType := getTypesStruct(t.Type); sType != nil {
				fmt.Printf("%s struct %v\n", fs.Position(node.Pos()), sType)
				pNum++
			}

		}
		return true
	})
	assert.Equal(t, pNum, 5)
}

func TestGetStructFields(t *testing.T) {
	src := `
package p

import "github.com/bigpigeon/toyorm"

type Product struct{
	toyorm.ModelDefault
	Data string
	Name string "toyorm:\"Alias:name\""
}

`
	fs := token.NewFileSet()
	parserFile, err := parser.ParseFile(fs, "src.go", src, 0)
	if err != nil {
		panic(err)
	}
	config := types.Config{Importer: importer.For("source", nil), FakeImportC: true}
	info := &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
		Scopes:     map[ast.Node]*types.Scope{},
		Defs:       make(map[*ast.Ident]types.Object),
	}
	//ast.Print(fs, parserFile)
	_, err = config.Check("./", fs, []*ast.File{parserFile}, info)
	if err != nil {
		t.Fatalf("checking package: %s", err)
	}
	ast.Inspect(parserFile, func(node ast.Node) bool {
		switch x := node.(type) {
		case *ast.StructType:
			tType := info.Types[x]
			if sType := getTypesStruct(tType.Type); sType != nil {
				fieldSet := []string{"ID", "CreatedAt", "UpdatedAt", "DeletedAt", "Data", "Name"}
				if fields := getStructFields(sType.Underlying().(*types.Struct)); len(fields) != 0 {
					assert.Equal(t, len(fields), len(fieldSet))
					for i := range fields {
						t.Log(fields[i])
						assert.Equal(t, fields[i].Name(), fieldSet[i])
					}
				}

				fieldAliasSet := []string{"ID", "CreatedAt", "UpdatedAt", "DeletedAt", "Data", "name"}
				fieldMap, err := getStructFieldMap(sType.Underlying().(*types.Struct), fs)
				assert.Nil(t, err)
				assert.Equal(t, len(fieldMap), len(fieldSet))
				for _, name := range fieldAliasSet {
					_, ok := fieldMap[name]
					assert.True(t, ok)
				}
			}

		}
		return true
	})
}
