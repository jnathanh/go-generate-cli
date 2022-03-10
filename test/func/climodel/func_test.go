package climodel_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCLIFromFunction(t *testing.T) {
	t.Run("hello func", func (t *testing.T)  {
		t.Cleanup(func() {os.Remove("../main.go")})

		// trigger generate

		cmd := exec.Command("go", "generate", "-x", "-run=Greet", ".")
		stderr := bytes.NewBuffer([]byte{})
		cmd.Stderr = stderr	
		out, err := cmd.Output()
		fmt.Println(string(out))
		fmt.Println(stderr)
		require.NoError(t, err)
	
		// for k, v := range map[string]string{
		// 	"GOARCH":    "arm64",
		// 	"GOOS":      "darwin",
		// 	"GOFILE":    "func.go",
		// 	"GOLINE":    "5",
		// 	"GOPACKAGE": "climodel",
		// 	"DOLLAR":    "$",
		// } {
		// 	require.NoError(t, os.Setenv(k, v))
		// }
		// require.NoError(t, lib.Exec())
	
		AssertCLIOutput(t, []string{"go", "run", "..", "Mr"}, "hello Mr\n")		
	})

	t.Run("dismiss func", func(t *testing.T) {
		t.Cleanup(func() {os.Remove("../main.go")})

		// trigger generate
		cmd := exec.Command("go", "generate", "-run=Dismiss", ".")
		out, err := cmd.Output()
		fmt.Println(string(out))
		require.NoError(t, err)
	
		AssertCLIOutput(t, []string{"go", "run", "..", "Mr"}, "goodbye Mr\n")		
	})

}

func AssertCLIOutput(t *testing.T, cmd []string, expectedOutput string) {
	t.Helper()
	c := exec.Command(cmd[0], cmd[1:]...)
	out, err := c.Output()
	if !assert.NoError(t, err) {
		t.Log(string(out))
		t.Fail()
	}
	assert.Equal(t, expectedOutput, string(out))
}