package main

import (
	"github.com/davecgh/go-spew/spew"
)

func main() {
	b := make([]byte, 10)
	b[0] = 2 >> 2
	spew.Dump(b)
}
