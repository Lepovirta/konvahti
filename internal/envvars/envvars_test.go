package envvars

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvVarsJSONUnmarshal(t *testing.T) {
	j := []byte(`
	{
		"MESSAGE": "hello",
		"A": "a",
		"B": "b"
	}
	`)

	var ev EnvVars
	err := json.Unmarshal(j, &ev)
	if !assert.NoError(t, err) {
		return
	}

	assert.ElementsMatch(
		t,
		EnvVars{
			"A=a",
			"B=b",
			"MESSAGE=hello",
		},
		ev,
	)
}
