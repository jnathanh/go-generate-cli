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

func testCLIFor(generateTag string, args []string, expectedOutput string, stdin io.Reader, debug bool) func(*testing.T) {
	return func(t *testing.T) {
		if !debug {
			t.Cleanup(func() {
				if !debug && !t.Failed() {
					// note that you may have to manually delete this file if it outputs invalid code before tests will run again (compile errors)
					os.Remove("main.go")
				}
			})
		}

		// trigger generate
		if !debug {
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

		AssertCLIOutput(t, stdin, append([]string{"go", "run", "."}, args...), expectedOutput)
	}
}

func TestGenerateCLIFromFunction(t *testing.T) {

	// this setup is only used by tests in debug mode, it's the best process I've found so far to debug from vs code
	// run in non-debug mode first then copy the values from the log for the desired test to this map
	for k, v := range map[string]string{
		"GOARCH":    "arm64",
		"GOOS":      "darwin",
		"GOFILE":    "func.go",
		"GOLINE":    "61",
		"GOPACKAGE": "main",
		"DOLLAR":    "$",
	} {
		require.NoError(t, os.Setenv(k, v))
	}

	t.Run("Greet", testCLIFor("Greet", []string{"Mr"}, "hello Mr\n", nil, false))
	t.Run("Dismiss", testCLIFor("Dismiss", []string{"Mr"}, "goodbye Mr\n", nil, false))
	t.Run("AddInts", testCLIFor("AddInts", []string{"1", "2"}, "3", nil, false))
	t.Run("AddInt8", testCLIFor("AddInt8", []string{"1", "2"}, "3", nil, false))
	t.Run("AddInt16", testCLIFor("AddInt16", []string{"1", "2"}, "3", nil, false))
	t.Run("AddInt32", testCLIFor("AddInt32", []string{"1", "2"}, "3", nil, false))
	t.Run("AddInt64", testCLIFor("AddInt64", []string{"1", "2"}, "3", nil, false))
	t.Run("EchoFloat32", testCLIFor("EchoFloat32", []string{"1.1"}, "1.1", nil, false))
	t.Run("EchoFloat64", testCLIFor("EchoFloat64", []string{"1.1"}, "1.1", nil, false))
	t.Run("BoolFlag", testCLIFor("BoolFlag", []string{"-on"}, "on", nil, false))
	t.Run("BoolFlag", testCLIFor("BoolFlag", []string{"--on"}, "on", nil, false))
	t.Run("BoolFlag", testCLIFor("BoolFlag", []string{}, "off", nil, false))
	t.Run("StdInOut", testCLIFor("StdInOut", nil, "ABC", strings.NewReader("abc"), false))
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
