/*
 * Copyright 2018. bigpigeon. All rights reserved.
 * Use of this source code is governed by a MIT style
 * license that can be found in the LICENSE file.
 */

package main

import (
	"github.com/bigpigeon/toyorm"
	"unsafe"
)

type Detail struct {
	ID        uint32
	ProductID uint32
	Data      string
}

type User struct {
	toyorm.ModelDefault
	Name string
	Sex  bool
}

type Product struct {
	toyorm.ModelDefault
	Name   string
	Detail Detail
	Users  []User
}

func NotMatch() {
	toy, err := toyorm.Open("sqlit3", "")

	if err != nil {
		panic(err)
	}
	// normal
	_ = toy.Model(&Product{}).Debug().Preload(unsafe.Offsetof(Product{}.Detail))
	_ = toy.Model(&Product{}).Debug().Preload("Detail")
	_ = toy.Model(&Product{}).Debug().OrderBy("UpdatedAt", "Name")
	_ = toy.Model(&Product{}).Debug().Where("=", unsafe.Offsetof(Product{}.Name), "pigeon")

	// field error
	_ = toy.Model(&Product{}).Debug().Preload("Name")
	_ = toy.Model(&Product{}).Debug().OrderBy("NotExistData", "NotExistTime")
	_ = toy.Model(&Product{}).Debug().Preload(unsafe.Offsetof(Product{}.Name))
	_ = toy.Model(&Product{}).Debug().Preload(unsafe.Offsetof(Detail{}.Data))
	//_ = toy.Model(&Product{}).Debug().Where("=", unsafe.Offsetof(Detail{}.Data), "pigeon")

	// join/enter
	_ = toy.Model(&Product{}).Debug().Preload(unsafe.Offsetof(Product{}.Users)).
		OrderBy(unsafe.Offsetof(User{}.Name)).Enter().
		OrderBy(unsafe.Offsetof(Product{}.ID))
	// join/enter field error
	_ = toy.Model(&Product{}).Debug().Preload(unsafe.Offsetof(Product{}.Users)).
		OrderBy(unsafe.Offsetof(Product{}.Name)).Enter().
		OrderBy(unsafe.Offsetof(User{}.ID))

	// assign indent test
	brick := toy.Model(&Product{})
	brick = brick.OrderBy(unsafe.Offsetof(Product{}.Name)).Preload(unsafe.Offsetof(Product{}.Detail))

	// indent field error
	brick.OrderBy(unsafe.Offsetof(Product{}.Name))
	brick2 := brick
	brick2.OrderBy(unsafe.Offsetof(Product{}.Name))
}
