package cli

import (
	"github.com/pkg/errors"
	"os"
	"strings"
)

type Spec struct {
	Params  []Value
	Output  Value
	Handler func(inputs Inputs) (interface{}, error)
}

func (s Spec) FlagArgSpec(name string) (v Value, exists bool) {
	for _, a := range s.Params {
		if strings.EqualFold(a.Name, name) {
			return v, true
		}
	}
	return v, false
}

type Inputs struct {
	Named map[string]interface{}
}

type Value struct {
	Name     string
	TypeName ValueType
}

type CLI struct {
	Spec
}

func New(s Spec) CLI {
	return CLI{
		Spec: s,
	}
}

func (cli CLI) Exec() error {
	unparsedArgs := []string{}
	if len(os.Args) > 1 {
		unparsedArgs = os.Args[1:]
	}

	// parse args
	parsed, err := cli.ParseArgs(unparsedArgs)
	if err != nil {
		return err
	}

	// validate args
	inputs, err := validateAndTypeArgs(parsed)
	if err != nil {
		return err
	}

	// delegate to handler
	output, err := cli.Handler(inputs)
	if err != nil {
		return err
	}

	// coerce output to stdout
	return writeToStdout(output)
}

func validateAndTypeArgs(args []ParsedArg) (i Inputs, err error) {
	for _, arg := range args {
		
	}
	return
}

func (cli CLI) ParseArgs(unparsedArgs []string) (args []ParsedArg, err error) {

	flagsTerminated := false
	for len(unparsedArgs) > 0 {
		// take one for parsing
		s := unparsedArgs[0]
		if len(unparsedArgs) > 1 {
			unparsedArgs = unparsedArgs[1:]
		} else {
			unparsedArgs = nil
		}

		a := ParseIsolatedArg(s, flagsTerminated)

		if a.FlagTerminator() {
			flagsTerminated = true
			continue
		}

		// add spec to arg
		argSpec, ok := cli.Spec.FlagArgSpec(a.Name)
		if !ok {
			// todo: also check ordered args
			return args, errors.Errorf("%q is not a valid flag")
		}
		a.Spec = argSpec

		// complete missing flag values
		if len(args) > 0 && !flagsTerminated {
			prev := args[len(args)-1]
			if prev.NamedArg() && prev.MissingValue() && !prev.Bool() && a.OrderedArg() {
				args[len(args)-1].Value = a.Value
				continue
			}
		}

		args = append(args, a)
	}
	return
}

type ValueType string

const (
	valueTypeBool ValueType = "bool"
)

type ParsedArg struct {
	Name  string
	Value string
	Spec  Value
}

func (a ParsedArg) OrderedArg() bool {
	return a.Name == ""
}

func (a ParsedArg) NamedArg() bool {
	return a.Name != ""
}

func (a ParsedArg) MissingValue() bool {
	return a.Value == ""
}

func (a ParsedArg) Bool() bool {
	return a.Spec.TypeName == valueTypeBool
}

func (a ParsedArg) FlagTerminator() bool {
	return a.Name == flagTerminator
}

const flagTerminator = "--"

func ParseIsolatedArg(s string, afterFlagTerminator bool) (a ParsedArg) {
	a.Value = s // default

	if afterFlagTerminator {
		return
	}

	// not a flag (no "-" prefix)
	if len(s) < 2 || s[0] != '-' {
		return
	}

	// flag terminator
	if s == flagTerminator {
		a.Name = flagTerminator
		return
	}

	// separate flag prefix from remainder
	flagPrefix := "-"
	if s[1] == '-' {
		flagPrefix = "--"
	}
	unprefixedFlag := strings.TrimPrefix(s, flagPrefix)

	// "--=" and "-=" are invalid flag formats, so it must be a positional arg
	if unprefixedFlag[0] == '=' {
		return
	}

	// try parse value with "=" format
	parts := strings.SplitN(unprefixedFlag, "=", 2)

	a.Name = parts[0]

	if len(parts) == 2 {
		a.Value = parts[1]
	}

	return
}
