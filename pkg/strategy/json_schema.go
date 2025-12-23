package strategy

import (
	"encoding/json"

	"github.com/invopop/jsonschema"
)

// ToJSONSchema converts a struct to a JSON schema
func ToJSONSchema[T any](t T) (string, error) {
	r := new(jsonschema.Reflector)
	r.DoNotReference = true
	schema := r.Reflect(t)

	jsonSchemaBytes, err := json.Marshal(schema)
	if err != nil {
		return "", err
	}

	return string(jsonSchemaBytes), nil
}
