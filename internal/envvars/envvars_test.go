package envvars

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestEnvVarsYAMLUnmarshal(t *testing.T) {
	j := []byte(`MESSAGE: hello
A: a
B: b
`)

	var ev EnvVars
	err := yaml.Unmarshal(j, &ev)
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
