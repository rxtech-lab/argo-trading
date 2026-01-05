package strategy

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/invopop/jsonschema"
)

// ToJSONSchema converts a struct to a JSON schema.
func ToJSONSchema[T any](t T) (string, error) {
	//nolint:exhaustruct // third-party struct with many optional fields
	r := &jsonschema.Reflector{
		DoNotReference: true,
		Mapper: func(t reflect.Type) *jsonschema.Schema {
			// Handle optional.Option[time.Time] as date-time string
			if strings.Contains(t.String(), "optional.Option[time.Time]") {
				//nolint:exhaustruct // third-party struct with many optional fields
				return &jsonschema.Schema{
					Type:   "string",
					Format: "date-time",
				}
			}

			return nil
		},
	}
	schema := r.Reflect(t)

	jsonSchemaBytes, err := json.Marshal(schema)
	if err != nil {
		return "", err
	}

	return string(jsonSchemaBytes), nil
}
