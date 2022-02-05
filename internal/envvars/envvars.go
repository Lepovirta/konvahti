package envvars

import (
	"encoding/json"
	"fmt"
	"strings"
)

type EnvVars []string

func (ev EnvVars) Add(key, value string) EnvVars {
	return append(ev, FromKeyValue(key, value)...)
}

func (ev EnvVars) Join(other EnvVars) EnvVars {
	if other == nil {
		return ev
	}
	return append(ev, other...)
}

func (ev EnvVars) Lookup(key string) (value string, ok bool) {
	for _, envVar := range ev {
		if strings.HasPrefix(envVar, key) {
			sepIndex := strings.Index(envVar, "=") + 1
			return envVar[sepIndex:], true
		}
	}
	return "", false
}

func FromKeyValue(key, value string) EnvVars {
	return EnvVars{fmt.Sprintf("%s=%s", key, value)}
}

func (ev *EnvVars) UnmarshalJSON(data []byte) error {
	var envMap map[string]string
	if err := json.Unmarshal(data, &envMap); err != nil {
		return err
	}
	for key, value := range envMap {
		*ev = append(*ev, FromKeyValue(key, value)...)
	}
	return nil
}
