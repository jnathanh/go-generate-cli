package lib

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path"
	"strconv"
	"text/template"

	"github.com/jnathanh/go-cli-generator/cli"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
)

func Exec() error {
	flag.Parse()

	fmt.Println("Generating CLI")

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Printf("  cwd = %s\n", cwd)
	fmt.Printf("  os.Args = %#v\n", os.Args)

	for _, ev := range []string{"GOARCH", "GOOS", "GOFILE", "GOLINE", "GOPACKAGE", "DOLLAR"} {
		fmt.Printf("%s=%q; ", ev, os.Getenv(ev))
	}
	fmt.Println()

	goGenerateLineNumber, err := strconv.Atoi(os.Getenv("GOLINE"))
	if err != nil {
		panic(err)
	}

	mode := packages.NeedName |
		packages.NeedFiles |
		packages.NeedCompiledGoFiles |
		packages.NeedImports |
		packages.NeedDeps |
		packages.NeedExportsFile |
		packages.NeedTypes |
		packages.NeedSyntax |
		packages.NeedTypesInfo |
		packages.NeedTypesSizes |
		packages.NeedModule

	packageName := os.Getenv("GOPACKAGE")
	sourceFile := path.Join(cwd, os.Getenv("GOFILE"))

	cfg := &packages.Config{Mode: mode}
	pkgs, err := packages.Load(cfg, flag.Args()...)
	if err != nil {
		return errors.WithStack(err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		return errors.WithStack(err)
	}

	// Lookup the annotated type

	// get syntax
	fileSyntax, targetPkg, err := GetFileAST(packageName, sourceFile, pkgs)
	if err != nil {
		return err
	}

	// get tokenized file
	tokenizedFile := GetTokenizedFile(sourceFile, targetPkg.Fset)
	if tokenizedFile == nil {
		return errors.New("could not find tokenized file")
	}

	var handlerCode string
	var spec cli.Spec
	for _, d := range fileSyntax.Decls {
		f, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}

		startLine := tokenizedFile.Line(f.Doc.Pos())
		endLine := tokenizedFile.Line(f.Doc.End())

		if goGenerateLineNumber < startLine || goGenerateLineNumber > endLine {
			continue
		}

		handlerCode, spec, err = FuncToHandlerAndSpec(f)
		if err != nil {
			return err
		}

		break
	}

	cliTemplate := `package main

import (
	"os"
	"fmt"
	"github.com/jnathanh/go-cli-generator/cli"
	"github.com/jnathanh/go-cli-generator/test/func/climodel"
)

func main() {
	spec := {{.Spec}}

	spec.Handler = {{.Handler}}

	cli := cli.New(spec)

	err := cli.Exec()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}
`

	type TemplateArgs struct {
		Spec    string
		Handler string
	}
	templateArgs := TemplateArgs{Spec: fmt.Sprintf("%#v", spec), Handler: handlerCode}
	tmpl, err := template.New("test").Parse(cliTemplate)
	if err != nil {
		return errors.WithStack(err)
	}

	cmdDirPath := path.Join(cwd, "..")
	mainPath := path.Clean(path.Join(cmdDirPath, "main.go"))

	fmt.Println("creating ", mainPath)
	f, err := os.Create(mainPath)
	if err != nil {
		return errors.WithStack(err)
	}

	buf := bytes.NewBuffer([]byte{})

	err = tmpl.Execute(buf, templateArgs)
	if err != nil {
		return errors.WithStack(err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		fmt.Println("generated code:\n", buf.String())
		return errors.WithStack(err)
	}

	_, err = io.Copy(f, bytes.NewBuffer(formatted))
	if err != nil {
		return errors.WithStack(err)
	}
	err = f.Close()
	if err != nil {
		return errors.WithStack(err)
	}

	fmt.Println("building to test for errors")
	outputPath := path.Join(cmdDirPath, "main")
	cmd := exec.Command("go", "build", "-o", outputPath, mainPath)
	stderr := bytes.NewBuffer([]byte{})
	cmd.Stderr = stderr
	out, err := cmd.Output()
	os.Remove(outputPath)
	fmt.Println(string(out))
	fmt.Println(stderr)

	// _, err = io.WriteString(f, handlerCode)
	return errors.WithStack(err)
}

func FuncToHandlerAndSpec(f *ast.FuncDecl) (handlerCode string, s cli.Spec, err error) {
	// add func params as cli params
	for _, p := range f.Type.Params.List {
		t := cli.ValueType(p.Type.(*ast.Ident).Name)
		s.Params = append(s.Params, cli.Value{
			Name:     p.Names[0].Name,
			TypeName: t,
			Ordered:  (t != cli.ValueTypeBool),
		})
	}

	// add the first return value as a cli output
	if len(f.Type.Results.List) > 0 {
		r := f.Type.Results.List[0]
		s.Output = cli.Value{
			Name:     r.Names[0].Name,
			TypeName: cli.ValueType(r.Type.(*ast.Ident).Name),
		}
	}

	// take interface types and type cast
	handlerCode = `func (inputs cli.Inputs) (output interface{}, err error) {
		name := inputs.Named["name"].(string)

		greeting := climodel.Greet(name)

		return greeting, nil
	}`

	// convert inputs to typed inputs

	// call handler with typed inputs
	// 	template = `package main

	// import (
	// 	"flag"
	// 	"io"
	// 	"os"

	// 	"github.com/jnathanh/go-cli-generator/test/func/cli"
	// )

	// func main() {
	// 	flag.Parse()

	// 	name := flag.Arg(0)

	// 	greeting := cli.Greet(name)

	// 	_, err := io.WriteString(os.Stdout, greeting)
	// 	if err == nil {
	// 		return
	// 	}

	// 	flag.Usage()
	// }
	// `
	return
}

func GetTokenizedFile(path string, fileSet *token.FileSet) (file *token.File) {
	fileSet.Iterate(func(f *token.File) (keepGoing bool) {
		if f.Name() != path {
			return true
		}
		file = f
		return false
	})
	return
}

func GetFileAST(packageName, filePath string, pkgs []*packages.Package) (fileSyntax *ast.File, targetPkg *packages.Package, err error) {
	// first find package
	for _, pkg := range pkgs {
		if pkg.Name != packageName {
			continue
		}
		targetPkg = pkg
	}

	if targetPkg == nil {
		return nil, nil, errors.Errorf("could not find package %q", packageName)
	}

	// then find file ast
	for i, path := range targetPkg.CompiledGoFiles {
		if path != filePath {
			continue
		}
		return targetPkg.Syntax[i], targetPkg, nil
	}

	return nil, targetPkg, errors.Errorf("found package %q, but did not find %q within it", packageName, filePath)
}

func LineEndPosition(lineNumber int, path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	defer func() {
		if err := scanner.Err(); err != nil {
			panic(err)
		}
	}()

	currentLine := 1
	byteCount := 0
	for scanner.Scan() {
		byteCount += len(scanner.Bytes()) + 1
		if currentLine == lineNumber {
			return byteCount, nil
		}
		currentLine++
	}

	return -1, errors.New("invalid line number")
}
