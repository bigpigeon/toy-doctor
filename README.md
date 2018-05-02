# Toy-Doctor

use to check toyorm syntax error in go project

### Install

    go get -u github.com/bigpigeon/toyorm

### Usage

    toy-doctor [directory]
    toy-doctor file1, file2 ...

### Example

some code in main.go
```golang
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
	if err := result.Err();err != nil {
		// sql error record it
	}
	// have error
	brick = brick.OrderBy(Offsetof(Detail{}.Name))
	result, err = brick.Find(&tab)
	if err != nil {
		panic(err)
	}
	if err := result.Err();err != nil {
		// sql error record it
	}
}
```

use toy-doctor to check it's error

    toy-doctor main.go
	// Output:
	// main.go:37:33 type must same as main.go:20:6
