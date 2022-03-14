package main_test

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/jnathanh/go-generate-cli/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCLIFor(generateTag string, c Case) func(*testing.T) {
	return func(t *testing.T) {
		if !c.Debug {
			t.Cleanup(func() {
				if !c.Debug && !t.Failed() {
					// note that you may have to manually delete this file if it outputs invalid code before tests will run again (compile errors)
					os.Remove("main.go")
				}
			})
		}

		// trigger generate
		if !c.Debug {
			cmd := exec.Command("go", "generate", "-x", fmt.Sprintf("-run=%s", generateTag), ".")
			out, err := cmd.CombinedOutput()
			t.Cleanup(func() {
				if t.Failed() {
					fmt.Println(string(out))
				}
			})
			require.NoError(t, err)
		} else {
			require.NoError(t, lib.Exec(lib.ExecOptions{Path: "main.go"}))
		}

		AssertCLIOutput(t, c.Stdin, append([]string{"go", "run", "."}, c.Args...), c.ExpectOut)
	}
}

type Case struct {
	Stdin     io.Reader
	Args      []string
	ExpectOut string
	Debug     bool
}

func TestGenerateCLIFromFunction(t *testing.T) {

	// this setup is only used by tests in debug mode, it's the best process I've found so far to debug from vs code
	// run in non-debug mode first then copy the values from the log for the desired test to this map
	for k, v := range map[string]string{
		"GOARCH":    "arm64",
		"GOOS":      "darwin",
		"GOFILE":    "func.go",
		"GOLINE":    "71",
		"GOPACKAGE": "main",
		"DOLLAR":    "$",
	} {
		require.NoError(t, os.Setenv(k, v))
	}

	t.Run("Greet", testCLIFor("Greet", Case{Args: []string{"Mr"}, ExpectOut: "hello Mr\n"}))
	t.Run("Dismiss", testCLIFor("Dismiss", Case{Args: []string{"Mr"}, ExpectOut: "goodbye Mr\n"}))
	t.Run("AddInts", testCLIFor("AddInts", Case{Args: []string{"1", "2"}, ExpectOut: "3"}))
	t.Run("AddInt8", testCLIFor("AddInt8", Case{Args: []string{"1", "2"}, ExpectOut: "3"}))
	t.Run("AddInt16", testCLIFor("AddInt16", Case{Args: []string{"1", "2"}, ExpectOut: "3"}))
	t.Run("AddInt32", testCLIFor("AddInt32", Case{Args: []string{"1", "2"}, ExpectOut: "3"}))
	t.Run("AddInt64", testCLIFor("AddInt64", Case{Args: []string{"1", "2"}, ExpectOut: "3"}))
	t.Run("EchoFloat32", testCLIFor("EchoFloat32", Case{Args: []string{"1.1"}, ExpectOut: "1.1"}))
	t.Run("EchoFloat64", testCLIFor("EchoFloat64", Case{Args: []string{"1.1"}, ExpectOut: "1.1"}))
	t.Run("BoolFlag", testCLIFor("BoolFlag", Case{Args: []string{"-on"}, ExpectOut: "on"}))
	t.Run("BoolFlag", testCLIFor("BoolFlag", Case{Args: []string{"--on"}, ExpectOut: "on"}))
	t.Run("BoolFlag", testCLIFor("BoolFlag", Case{Args: []string{}, ExpectOut: "off"}))
	t.Run("StdInOut", testCLIFor("StdInOut", Case{Stdin: strings.NewReader("abc"), ExpectOut: "ABC"}))
	t.Run("NoInputsOrOutputs", testCLIFor("NoInputsOrOutputs", Case{}))
}

func AssertCLIOutput(t *testing.T, stdin io.Reader, cmd []string, expectedOutput string) {
	t.Helper()
	c := exec.Command(cmd[0], cmd[1:]...)

	if stdin != nil {
		c.Stdin = stdin
	}

	out, err := c.CombinedOutput()
	t.Cleanup(func() {
		if t.Failed() {
			fmt.Println(string(out))
		}
	})
	require.NoError(t, err)
	assert.Equal(t, expectedOutput, string(out))
}
