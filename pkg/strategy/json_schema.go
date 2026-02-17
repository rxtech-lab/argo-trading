package strategy

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/invopop/jsonschema"
)

// GetKeychainFields returns the JSON field names of struct fields tagged with `keychain:"true"`.
func GetKeychainFields[T any](t T) []string {
	var fields []string

	v := reflect.TypeOf(t)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fields
	}

	for i := range v.NumField() {
		field := v.Field(i)
		if field.Tag.Get("keychain") == "true" {
			name := field.Tag.Get("json")
			if name == "" {
				name = field.Name
			} else {
				// Strip ",omitempty" or other options
				if idx := strings.Index(name, ","); idx != -1 {
					name = name[:idx]
				}
			}

			fields = append(fields, name)
		}
	}

	return fields
}

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
