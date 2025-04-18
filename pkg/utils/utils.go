package utils

import (
	"encoding/json"

	"github.com/invopop/jsonschema"
)

func GetSchemaFromConfig(config any) (string, error) {
	schema := jsonschema.Reflect(config)

	jsonSchemaBytes, err := json.Marshal(schema)
	if err != nil {
		return "", err
	}

	return string(jsonSchemaBytes), nil
}
