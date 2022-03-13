package cli_test

import (
	"testing"

	"github.com/jnathanh/go-cli-generator/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoolFlagOmitted(t *testing.T) {
	spec := cli.Spec{
		Params: []cli.Value{
			{Name: "on", TypeName: "bool", Ordered: false},
		},
		Handler: func(inputs cli.Inputs) (output interface{}, err error) {

			on, ok := inputs.Named["on"].(bool)

			assert.True(t, ok)
			assert.False(t, on)

			return nil, nil
		},
	}

	require.NoError(t, cli.New(spec).ExecArgs([]string{""}))
}
