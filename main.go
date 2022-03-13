package main

import (
	"flag"
	"fmt"

	"github.com/jnathanh/go-cli-generator/lib"
)
func main() {
	// this is a workaround to make testing more convenient, the only behavior it changes is avoiding errors when this flag is passed
	flag.String("testsignature", "","flag for testing, does nothing except not throw an error when a value is passed with it")	
	pathArg := flag.String("path", "main.go", "specifies which path the generated main.go file should be written to")
	flag.Parse()

	err := lib.Exec(lib.ExecOptions{Path: *pathArg})
	if err != nil {
		fmt.Printf("%+v\n", err)
	}
}

