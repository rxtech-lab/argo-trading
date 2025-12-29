package version

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// CheckVersionCompatibility checks if engine and runtime versions are compatible.
// Returns nil if compatible, error with details if not.
//
// Compatibility Rules:
//   - If either version is "main" (development build), compatibility check is skipped
//   - Major versions must match exactly
//   - Minor versions must match exactly
//   - Patch versions can differ (e.g., 1.2.0 is compatible with 1.2.5)
//
// Examples:
//   - Engine 1.2.0, Runtime 1.2.0 -> OK (exact match)
//   - Engine 1.2.1, Runtime 1.2.0 -> OK (patch differs)
//   - Engine 1.3.0, Runtime 1.2.0 -> ERROR (minor differs)
//   - Engine 2.0.0, Runtime 1.2.0 -> ERROR (major differs)
//   - Engine main, Runtime 1.2.0 -> OK (dev build, skip check)
//   - Engine 1.2.0, Runtime main -> OK (dev build, skip check)
func CheckVersionCompatibility(engineVersion, runtimeVersion string) error {
	// Strip 'v' prefix if present for consistency
	engineVersion = strings.TrimPrefix(engineVersion, "v")
	runtimeVersion = strings.TrimPrefix(runtimeVersion, "v")

	// Skip version check for "main" (development builds)
	if engineVersion == "main" || runtimeVersion == "main" {
		return nil
	}

	// Parse engine version
	engineSemver, err := semver.NewVersion(engineVersion)
	if err != nil {
		return fmt.Errorf("invalid engine version '%s': %w", engineVersion, err)
	}

	// Parse runtime version
	runtimeSemver, err := semver.NewVersion(runtimeVersion)
	if err != nil {
		return fmt.Errorf("invalid runtime version '%s': %w", runtimeVersion, err)
	}

	// Check major version match
	if engineSemver.Major() != runtimeSemver.Major() {
		return fmt.Errorf("major version mismatch: engine is %d.x.x but strategy requires %d.x.x",
			engineSemver.Major(), runtimeSemver.Major())
	}

	// Check minor version match
	if engineSemver.Minor() != runtimeSemver.Minor() {
		return fmt.Errorf("minor version mismatch: engine is %d.%d.x but strategy requires %d.%d.x",
			engineSemver.Major(), engineSemver.Minor(),
			runtimeSemver.Major(), runtimeSemver.Minor())
	}

	// Patch versions can differ, so we're compatible
	return nil
}
