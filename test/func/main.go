package main

import (
	"fmt"
	"github.com/jnathanh/go-cli-generator/cli"
	"github.com/jnathanh/go-cli-generator/test/func/climodel"
	"io"
	"os"
)

func main() {
	spec := cli.Spec{Params: []cli.Value{cli.Value{Name: "name", TypeName: "string"}}, Output: cli.Value{Name: "greeting", TypeName: "string"}, Handler: (func(cli.Inputs) error)(nil)}
	spec.Handler = func(inputs cli.Inputs) error {
		name := inputs.Named["name"].(string)

		greeting := climodel.Greet(name)

		_, err := io.WriteString(os.Stdout, greeting)
		if err != nil {
			return err
		}

		return nil
	}

	cli := cli.New(spec)

	err := cli.Exec()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}
