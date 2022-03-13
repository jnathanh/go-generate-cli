go-generate-cli is a proof-of-concept project testing the idea "what if you could define a command line interface the same way you define a function?" Any given command in a command line interface (CLI) is just some logic that is executed with inputs and produces outputs, so why make defining that action more complicated than a simple function definition?

Example usage:

```go
package main

//go:generate go-generate-cli
func Greet(name string) (greeting string) {
	return "hello " + name + "\n"
}
```

```sh
# in the same package folder as the Greet function (or from a parent folder with ./... arg)
go generate
go build -o greet .
./greet John
# outputs "hello John"
```

## Learnings and next steps

- It's possible to auto-create a CLI with a function as a template.
- Having generated code handle the actual implementation does make it easier to simply define a CLI.
- To realize the simplicity the generator needs to be very stable and handle all common use cases and output clear errors when attempted outside those use cases.
- Unfortunately, the actual controller of the CLI is the generated code, so changes to the CLI model (function template) will only be reflected after re-generating and rebuilding. Also the generated code is now in your repo and is clearly the source-of-truth as the CLI controller. Alternatively you could use a process that generates -> builds -> and deletes the generated code. That would leave the function template as the reference code for the CLI.
- Try generating a resourceful CLI (similar to aws cli with n levels of resource types and subtypes with actions at the leaves of the resource tree) using a struct as a template and functions as fields (or methods) as the CLI action handlers.
- Improve CLI features
    - more types
    - add auto-documentation
    - add customization options
    - custom documentation via code comments
- Try alternate implementation that relies more heavily on reflection rather than code generation to implement the CLI (to perhaps make the CLI implementation wrapping more transparent/not obscure the actual CLI model template)
