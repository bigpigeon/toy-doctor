/*
 * Copyright 2018. bigpigeon. All rights reserved.
 * Use of this source code is governed by a MIT style
 * license that can be found in the LICENSE file.
 */

package main

// run likes: toy-doctor exampledata/
func ExampleCheck() {
	args := []string{
		"exampledata/",
	}
	Main(args)
	// Output:
	// exampledata/main.go:37:33 type must same as exampledata/main.go:20:6
}
