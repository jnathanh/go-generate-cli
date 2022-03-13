package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jnathanh/go-generate-cli/lib"
)

func main() {
	// this is a workaround to make testing more convenient, the only behavior it changes is avoiding errors when this flag is passed
	flag.String("testsignature", "", "flag for testing, does nothing except not throw an error when a value is passed with it")
	pathArg := flag.String("path", "main.go", "specifies which path the generated main.go file should be written to")
	flag.Parse()

	if l := os.Getenv("GOLINE"); l == "" {
		fmt.Println("currently supported only via invocation by `go generate`\nadd a `//go:generate go-generate-cli` line above the function you wish to be the model for the command line interface you want to generate")
		os.Exit(1)
	}

	err := lib.Exec(lib.ExecOptions{Path: *pathArg})
	if err != nil {
		fmt.Printf("%+v\n", err)
	}
}
