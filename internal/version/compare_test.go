package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckVersionCompatibility(t *testing.T) {
	tests := []struct {
		name           string
		engineVersion  string
		runtimeVersion string
		expectError    bool
		errorContains  string
	}{
		// Compatible cases (from the requirements table)
		{
			name:           "exact match",
			engineVersion:  "1.2.0",
			runtimeVersion: "1.2.0",
			expectError:    false,
		},
		{
			name:           "engine patch higher",
			engineVersion:  "1.2.1",
			runtimeVersion: "1.2.0",
			expectError:    false,
		},
		{
			name:           "runtime patch higher",
			engineVersion:  "1.2.0",
			runtimeVersion: "1.2.5",
			expectError:    false,
		},
		{
			name:           "same major minor different patch",
			engineVersion:  "2.5.10",
			runtimeVersion: "2.5.3",
			expectError:    false,
		},

		// Incompatible cases (from the requirements table)
		{
			name:           "engine minor higher",
			engineVersion:  "1.3.0",
			runtimeVersion: "1.2.0",
			expectError:    true,
			errorContains:  "minor version mismatch",
		},
		{
			name:           "engine minor lower",
			engineVersion:  "1.1.0",
			runtimeVersion: "1.2.0",
			expectError:    true,
			errorContains:  "minor version mismatch",
		},
		{
			name:           "major version differs",
			engineVersion:  "2.0.0",
			runtimeVersion: "1.2.0",
			expectError:    true,
			errorContains:  "major version mismatch",
		},
		{
			name:           "engine is main",
			engineVersion:  "main",
			runtimeVersion: "1.2.0",
			expectError:    false,
		},
		{
			name:           "engine is main with different runtime",
			engineVersion:  "main",
			runtimeVersion: "1.3.0",
			expectError:    false,
		},
		{
			name:           "both are main",
			engineVersion:  "main",
			runtimeVersion: "main",
			expectError:    false,
		},
		{
			name:           "runtime is main",
			engineVersion:  "1.2.0",
			runtimeVersion: "main",
			expectError:    false,
		},

		// Edge cases with v prefix
		{
			name:           "v prefix on engine",
			engineVersion:  "v1.2.0",
			runtimeVersion: "1.2.0",
			expectError:    false,
		},
		{
			name:           "v prefix on runtime",
			engineVersion:  "1.2.0",
			runtimeVersion: "v1.2.0",
			expectError:    false,
		},
		{
			name:           "v prefix on both",
			engineVersion:  "v1.2.0",
			runtimeVersion: "v1.2.0",
			expectError:    false,
		},

		// Edge cases with prerelease and metadata
		{
			name:           "prerelease version",
			engineVersion:  "1.2.0-alpha",
			runtimeVersion: "1.2.0",
			expectError:    false,
		},
		{
			name:           "build metadata",
			engineVersion:  "1.2.0+build123",
			runtimeVersion: "1.2.0",
			expectError:    false,
		},

		// Invalid versions
		{
			name:           "invalid engine version",
			engineVersion:  "not-a-version",
			runtimeVersion: "1.2.0",
			expectError:    true,
			errorContains:  "invalid engine version",
		},
		{
			name:           "invalid runtime version",
			engineVersion:  "1.2.0",
			runtimeVersion: "not-a-version",
			expectError:    true,
			errorContains:  "invalid runtime version",
		},
		{
			name:           "empty engine version",
			engineVersion:  "",
			runtimeVersion: "1.2.0",
			expectError:    true,
			errorContains:  "invalid engine version",
		},
		{
			name:           "empty runtime version",
			engineVersion:  "1.2.0",
			runtimeVersion: "",
			expectError:    true,
			errorContains:  "invalid runtime version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckVersionCompatibility(tt.engineVersion, tt.runtimeVersion)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetVersion(t *testing.T) {
	v := GetVersion()
	assert.Equal(t, Version, v)
}
