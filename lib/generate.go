package lib

import (
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
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/jnathanh/go-cli"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
)

type ExecOptions struct {
	Path string
}

func Exec(o ExecOptions) error {
	if o.Path == "" {
		o.Path = "main.go"
	}
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

	cfg := &packages.Config{Mode: mode, Tests: true}
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

	decl, err := funcDecl(fileSyntax.Decls, tokenizedFile, goGenerateLineNumber)
	if err != nil {
		return err
	}
	
	handlerCode, spec, pkgPaths, err := FuncToHandlerAndSpec(decl, targetPkg)
	if err != nil {
		return err
	}

	cliTemplate := `package main

import (
	"os"
	"fmt"
	"github.com/jnathanh/go-cli"
	{{ range .PkgPaths }}
		{{ if . }}"{{ . }}"{{ end }}
	{{ end }}
)

func main() {
	spec := {{.Spec}}

	spec.Handler = {{.Handler}}

	cli := cli.New(spec)

	err := cli.ExecOSArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v", err)
		os.Exit(1)
	}
}
`

	type TemplateArgs struct {
		Spec     string
		Handler  string
		PkgPaths []string
	}

	templateArgs := TemplateArgs{
		Spec:     fmt.Sprintf("%#v", spec),
		Handler:  handlerCode,
		PkgPaths: pkgPaths,
	}

	tmpl, err := template.New("test").Parse(cliTemplate)
	if err != nil {
		return errors.WithStack(err)
	}

	p, err := filepath.Abs(o.Path)
	if err != nil {
		return err
	}

	cmdDirPath, _ := path.Split(p)
	// cmdDirPath := path.Join(cwd, "..")
	// mainPath := path.Clean(path.Join(cmdDirPath, "main.go"))

	if strings.TrimSuffix(cmdDirPath, "/") != cwd {
		templateArgs.PkgPaths = append(templateArgs.PkgPaths, targetPkg.PkgPath)
	}

	fmt.Println("creating ", p)
	f, err := os.Create(p)
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
	cmd := exec.Command("go", "build", "-o", outputPath, cmdDirPath)
	fmt.Println(cmd.String())
	out, err := cmd.CombinedOutput()
	os.Remove(outputPath)
	fmt.Println(string(out))

	// _, err = io.WriteString(f, handlerCode)
	return errors.WithStack(err)
}

func funcDecl(decls []ast.Decl, tokenizedFile *token.File, goGenerateLineNumber int) (*ast.FuncDecl, error) {
	for _, d := range decls {
		f, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}

		startLine := tokenizedFile.Line(f.Doc.Pos())
		endLine := tokenizedFile.Line(f.Doc.End())

		if goGenerateLineNumber < startLine || goGenerateLineNumber > endLine {
			continue
		}

		return f, nil
	}
	return nil, errors.Errorf("unable to find function declaration with comment lines on line %d in file %q", goGenerateLineNumber, tokenizedFile.Name())
}

type Type struct {
	TypeName    string
	PackageName string
	PackagePath string
}

func (t *Type) IsReader() bool {
	return t.PackagePath == "io" && t.TypeName == "Reader"
}

type valueAttrs struct {
	Name string
	Type
}	

func fieldType(f *ast.Field, pkg *packages.Package) (t Type, err error) {
	switch v := f.Type.(type) {
	case *ast.SelectorExpr:
		t.TypeName = v.Sel.Name
		t.PackageName = v.X.(*ast.Ident).Name

		p, ok := pkg.Imports[t.PackageName]
		if !ok {
			return t, errors.Errorf("cannot find import for %s.%s in package %s", t.PackageName, t.TypeName, pkg.Name)
		}
		t.PackagePath = p.PkgPath
	case *ast.Ident:
		t.TypeName = v.Name
	default:
		err = errors.Errorf("unhandled func param type: %T", f.Type)
	}
	return
}

func FuncToHandlerAndSpec(f *ast.FuncDecl, pkg *packages.Package) (handlerCode string, s cli.Spec, pkgPaths []string, err error) {
	type tmplArgs struct {
		Params      []valueAttrs
		Output      valueAttrs
		HandlerName string
		HandlerPkg  string
	}

	args := tmplArgs{
		HandlerName: f.Name.Name,
		HandlerPkg:  pkg.Name,
	}

	pkgMap := map[string]bool{}

	// add func params as cli params
	for _, p := range f.Type.Params.List {
		t, err := fieldType(p, pkg)
		if err != nil {
			return "", s, pkgPaths, err
		}

		if !t.IsReader() { // this is interpreted as os.Stdin, and "os" is already imported
			pkgMap[t.PackagePath] = true
		}

		for _, n := range p.Names {
			args.Params = append(args.Params, valueAttrs{
				Name: n.Name,
				Type: t,
			})

			s.Params = append(s.Params, cli.Value{
				Name:     n.Name,
				TypeName: cli.ValueType(t.TypeName),
				Ordered:  (cli.ValueType(t.TypeName) != cli.ValueTypeBool),
			})
		}
	}

	// add the first return value as a cli output
	if len(f.Type.Results.List) > 0 {

		r := f.Type.Results.List[0]

		t, err := fieldType(r, pkg)
		if err != nil {
			return "", s, pkgPaths, err
		}

		args.Output = valueAttrs{Name: r.Names[0].Name, Type: t}
		s.Output = cli.Value{
			Name:     r.Names[0].Name,
			TypeName: cli.ValueType(t.TypeName),
		}
	}

	// take interface types and type cast
	funcTmpl := `func (inputs cli.Inputs) (output interface{}, err error) {

		{{ range .Params }}
			{{ if not .IsReader}}{{ .Name }} := inputs.Named["{{ .Name }}"].({{ if .PackageName }}{{ .PackageName }}.{{ end }}{{ .TypeName }}){{ end }}
		{{ end }}

		{{ .Output.Name }} := {{ if ne .HandlerPkg "main" }}{{ .HandlerPkg }}.{{ end }}{{ .HandlerName }}({{ range $i, $p := .Params }}{{ if $i }} ,{{ end }}{{ if $p.IsReader }}os.Stdin{{ else }}{{ $p.Name }}{{ end }}{{ end }})

		return {{ .Output.Name }}, nil
	}`

	tmpl, err := template.New("test").Parse(funcTmpl)
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	b := bytes.NewBuffer([]byte{})
	err = errors.WithStack(tmpl.Execute(b, args))
	handlerCode = b.String()

	for k, _ := range pkgMap {
		pkgPaths = append(pkgPaths, k)
	}

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
	var packagesWithMatchingName []*packages.Package
	for _, pkg := range pkgs {
		if pkg.Name != packageName {
			continue
		}

		packagesWithMatchingName = append(packagesWithMatchingName, pkg)
		// targetPkg = pkg
	}

	if len(packagesWithMatchingName) == 0 {
		return nil, nil, errors.Errorf("could not find package %q", packageName)
	}

	// then find file ast
	for _, pkg := range packagesWithMatchingName {
		for i, path := range pkg.CompiledGoFiles {
			if path != filePath {
				continue
			}
			return pkg.Syntax[i], pkg, nil
		}
	}

	return nil, targetPkg, errors.Errorf("found package %q, but did not find %q within it", packageName, filePath)
}
