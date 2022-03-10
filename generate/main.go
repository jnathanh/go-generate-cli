package main

import (
	"flag"
	"fmt"

	"github.com/jnathanh/go-cli-generator/generate/lib"
)
func main() {
	// this is a workaround to make testing more convenient, the only behavior it changes is avoiding errors when this flag is passed
	flag.String("testsignature", "","")
	flag.Parse()

	err := lib.Exec()
	if err != nil {
		fmt.Printf("%+v\n", err)
	}
}

