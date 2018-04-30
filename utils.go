/*
 * Copyright 2018. bigpigeon. All rights reserved.
 * Use of this source code is governed by a MIT style
 * license that can be found in the LICENSE file.
 */

package main

import (
	"errors"
	"fmt"
	"github.com/bigpigeon/toyorm"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"strings"
)

func getTypesStruct(_type types.Type) *types.Named {
	switch x := _type.(type) {
	case *types.Slice:
		return getTypesStruct(x.Elem())
	case *types.Pointer:
		return getTypesStruct(x.Elem())
	case *types.Named:
		if _, ok := x.Underlying().(*types.Struct); ok {
			return x
		}
	}
	return nil
}

// get ident for selector.Sel or ident
func getIdent(expr ast.Expr) *ast.Ident {
	switch x := expr.(type) {
	case *ast.SelectorExpr:
		return x.Sel
	case *ast.Ident:
		return x
	}
	return nil
}

func getStructFields(structType *types.Struct) []*types.Var {
	var fields []*types.Var
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		if field.Anonymous() {
			if subStructType, ok := field.Type().Underlying().(*types.Struct); ok {
				if subFields := getStructFields(subStructType); len(subFields) != 0 {
					fields = append(fields, subFields...)
				}
			}
		} else {
			fields = append(fields, field)
		}
	}
	return fields
}

func getStructFieldMap(structType *types.Struct, fs *token.FileSet) (map[string]*types.Var, error) {
	m := map[string]*types.Var{}
MainRange:
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		if field.Anonymous() {
			if subStructType, ok := field.Type().Underlying().(*types.Struct); ok {
				if subMap, err := getStructFieldMap(subStructType, fs); err == nil {
					for k, v := range subMap {
						if _, ok := m[k]; ok {
							return nil, errors.New(fs.Position(v.Pos()).String() + " duplicate key ")
						}
						m[k] = v
					}
				} else {
					return nil, err
				}
			}
		} else {
			tag := reflect.StructTag(structType.Tag(i))
			toyormTag := tag.Get("toyorm")
			keyValuList := toyorm.GetTagKeyVal(toyormTag)
			for _, keyVal := range keyValuList {
				if keyVal.Key == "alias" {
					m[keyVal.Val] = field
					continue MainRange
				}
			}
			m[field.Name()] = field
		}
	}
	return m, nil
}

type TypesStructList []*types.Named

func (l TypesStructList) Copy() TypesStructList {
	r := make(TypesStructList, len(l))
	copy(r, l)
	return r
}

func standardSrc() string {
	brickType := reflect.TypeOf(&toyorm.ToyBrick{})
	src := `
package main
import "github.com/bigpigeon/toyorm"
import "unsafe"

type Product struct{
	toyorm.ModelDefault
	Data string
}
func main(){
	toy, err := toyorm.Open("sqlite3", "")
	if err != nil {
		panic(err)
	}
	// this usage with toyorm is error, don't try to use it
	
	// toy model
	brick := toy.Model(&Product{})
	// chain funcs
	{
		%s
	}
	// preload
	_ = brick.Preload
	// enter
	_ = brick.Enter
	// join
	_ = brick.Join
	// Swap
	_ = brick.Swap
	// offsetof
	_ = unsafe.Offsetof(Product{}.ID)
	
}
`
	var methodList []string
	for i := 0; i < brickType.NumMethod(); i++ {
		method := brickType.Method(i)
		if method.Type.NumOut() == 1 && method.Type.Out(0) == brickType {
			switch method.Name {
			case "Preload", "Enter", "Join", "Swap":
			default:
				methodList = append(methodList, fmt.Sprintf("_ = brick.%s", method.Name))
			}
		}
	}
	methodList = append(methodList, "_ = brick.Or")
	methodList = append(methodList, "_ = brick.And")
	src = fmt.Sprintf(src, strings.Join(methodList, "\n\t\t"))
	return src
}
