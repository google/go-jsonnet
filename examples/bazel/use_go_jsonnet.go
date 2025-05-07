package main

import (
	"fmt"

	gjs "github.com/google/go-jsonnet"
)

func main() {
	fmt.Printf("Example using go jsonnet (%s)\n", gjs.Version())

	vm := gjs.MakeVM()
	out, err := vm.EvaluateFile("example.jsonnet")
	if err != nil {
		fmt.Printf("%v\n", err)
	} else {
		fmt.Printf("%s", out)
	}
}
