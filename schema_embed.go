package tfyt

import _ "embed"

//go:embed tfyt.schema.json
var configSchema []byte

func ConfigSchema() []byte {
	return configSchema
}
