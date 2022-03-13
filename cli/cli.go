package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type Spec struct {
	Params  []Value
	Output  Value
	Handler func(inputs Inputs) (interface{}, error)
}

func (s Spec) FlagArgSpec(name string) (v Value, exists bool) {
	for _, p := range s.Params {
		if strings.EqualFold(p.Name, name) {
			return p, true
		}
	}
	return
}

// OrderedArgSpec returns the specification for an ordered argument (0 based index)
func (s Spec) OrderedArgSpec(position int) (v Value, exists bool) {
	i := 0
	for _, p := range s.Params {
		if !p.Ordered {
			continue
		}

		i++

		if (i - 1) == position {
			return p, true
		}
	}
	return v, false
}

type Inputs struct {
	Named map[string]interface{}
}

func (i Inputs) AddArg(name string, value string, t ValueType) error {
	if vPrev, ok := i.Named[name]; ok {
		return errors.Errorf("argument %q has been provided more than one time: %q, %q", name, vPrev, value)
	}

	// coerce value to type
	v, err := stringToType(value, t)
	if err != nil {
		return err
	}

	// save to inputs
	i.Named[name] = v

	return nil
}

type Value struct {
	Name     string
	TypeName ValueType
	Ordered  bool
}

type CLI struct {
	Spec
}

func New(s Spec) CLI {
	return CLI{
		Spec: s,
	}
}

func (cli CLI) ExecOSArgs() error {
	return cli.ExecArgs(os.Args)
}

func (cli CLI) ExecArgs(args []string) error {
	unparsedArgs := []string{}
	if len(args) > 1 {
		unparsedArgs = args[1:]
	}

	// parse args
	parsed, err := cli.ParseArgs(unparsedArgs)
	if err != nil {
		return err
	}

	// validate args
	inputs, err := validateAndTypeArgs(parsed, cli.Spec)
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

func writeToStdout(output interface{}) error {
	if output == nil {
		return nil
	}

	r, err := anyToReader(output)
	if err != nil {
		return err
	}

	_, err = io.Copy(os.Stdout, r)

	return err
}

func anyToReader(i interface{}) (io.Reader, error) {
	if r, ok := i.(io.Reader); ok {
		return r, nil
	}

	switch v := i.(type) {
	case string:
		return strings.NewReader(v), nil
	case int, int8, int16, int32, int64:
		return strings.NewReader(fmt.Sprintf("%d", v)), nil
	case float32:
		return strings.NewReader(strconv.FormatFloat(float64(v), 'f', -1, 32)), nil
	case float64:
		return strings.NewReader(strconv.FormatFloat(v, 'f', -1, 64)), nil
	case bool:
		return strings.NewReader(strconv.FormatBool(v)), nil
	default:
		return nil, errors.Errorf("unable to convert type %T to io.Reader (value=%v)", i, i)
	}
}

func validateAndTypeArgs(args []ParsedArg, spec Spec) (i Inputs, err error) {
	if i.Named == nil {
		i.Named = map[string]interface{}{}
	}

	orderedArgs := []ParsedArg{}
	for _, arg := range args {
		if arg.OrderedArg() {
			argIndex := len(orderedArgs)

			s, exists := spec.OrderedArgSpec(argIndex)
			if !exists {
				return i, errors.Errorf("no argument defined for argument position %d, (given %q)\n", argIndex+1, arg.Value)
			}

			// todo: is this even needed?
			arg.Spec = s

			// track the order (for the next ordered arg position)
			orderedArgs = append(orderedArgs, arg)

			// add to results
			err = i.AddArg(arg.Spec.Name, arg.Value, s.TypeName)
			if err != nil {
				return
			}

			continue
		}

		// flag

		// should already have this, but check just in case
		if (arg.Spec == Value{}) {
			s, exists := spec.FlagArgSpec(arg.Name)
			if !exists {
				return i, errors.Errorf("no argument defined for flag %q, (given %q)\n", arg.Name, arg.Value)
			}

			arg.Spec = s
		}

		// add to results
		err = i.AddArg(arg.Spec.Name, arg.Value, arg.Spec.TypeName)
		if err != nil {
			return
		}
	}

	// add missing bool params (as false)
	for _, p := range spec.Params {
		if p.TypeName != ValueTypeBool {
			continue
		}

		if _, exists := i.Named[p.Name]; exists {
			continue
		}

		i.Named[p.Name] = false
	}

	return
}

func stringToType(s string, t ValueType) (interface{}, error) {
	switch t {
	case ValueTypeBool:
		// only flags can be value types, and their presence always means true (absence == false)
		return true, nil
	case ValueTypeString:
		return s, nil
	case ValueTypeInt:
		return strconv.Atoi(s)
	case ValueTypeInt8:
		i, err := strconv.ParseInt(s, 10, 8)
		return int8(i), err
	case ValueTypeInt16:
		i, err := strconv.ParseInt(s, 10, 16)
		return int16(i), err
	case ValueTypeInt32:
		i, err := strconv.ParseInt(s, 10, 32)
		return int32(i), err
	case ValueTypeInt64:
		i, err := strconv.ParseInt(s, 10, 64)
		return int64(i), err
	case ValueTypeFloat32:
		f, err := strconv.ParseFloat(s, 32)
		return float32(f), err
	case ValueTypeFloat64:
		f, err := strconv.ParseFloat(s, 64)
		return float64(f), err
	default:
		return nil, errors.Errorf("no support for parsing arg type %q yet\n", t)
	}
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
		a.Spec, _ = cli.Spec.FlagArgSpec(a.Name)

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
	ValueTypeBool    ValueType = "bool"
	ValueTypeString  ValueType = "string"
	ValueTypeInt     ValueType = "int"
	ValueTypeInt8    ValueType = "int8"
	ValueTypeInt16   ValueType = "int16"
	ValueTypeInt32   ValueType = "int32"
	ValueTypeInt64   ValueType = "int64"
	ValueTypeFloat32 ValueType = "float32"
	ValueTypeFloat64 ValueType = "float64"

	// how to implement this?
	ValueTypeReader ValueType = "reader"
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
	return a.Spec.TypeName == ValueTypeBool
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
