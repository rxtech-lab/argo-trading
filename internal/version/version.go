package version

// Version is the current version of the argo-trading library.
// This value is set at build time using ldflags:
// -ldflags "-X github.com/rxtech-lab/argo-trading/internal/version.Version=1.2.3"
// The default value "main" indicates a development build.
var Version = "v1.11.0"

// GetVersion returns the current version of the library.
func GetVersion() string {
	return Version
}
