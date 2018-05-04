/*
 * Copyright 2018. bigpigeon. All rights reserved.
 * Use of this source code is governed by a MIT style
 * license that can be found in the LICENSE file.
 */

package main

import (
	"github.com/bigpigeon/toyorm"
	. "unsafe"
)

type Detail struct {
	ID        uint32 `toyorm:"primary key;auto_increment"`
	ProductID uint32 `toyorm:"index"`
	Name      string
}

type Product struct {
	toyorm.ModelDefault
	Name   string `toyorm:"index"`
	Detail *Detail
}

func Foo(brick *toyorm.ToyBrick) {
	brick.OrderBy(Offsetof(Product{}.CreatedAt)).Preload(Offsetof(Product{}.Detail)).Enter()
	var tab []Product
	result, err := brick.Find(&tab)
	if err != nil {
		panic(err)
	}
	if err := result.Err(); err != nil {
		// sql error record it
	}
}

func main() {
	toy, err := toyorm.Open("sqlite3", "")
	if err != nil {
		panic(err)
	}
	brick := toy.Model(&Product{})
	// to preload detail
	brick = brick.OrderBy(Offsetof(Product{}.CreatedAt)).Preload(Offsetof(Product{}.Detail)).Enter()
	var tab []Product
	result, err := brick.Find(&tab)
	if err != nil {
		panic(err)
	}
	if err := result.Err(); err != nil {
		// sql error record it
	}
	// have error
	brick = brick.OrderBy(Offsetof(Detail{}.Name))
	result, err = brick.Find(&tab)
	if err != nil {
		panic(err)
	}
	if err := result.Err(); err != nil {
		// sql error record it
	}
}
