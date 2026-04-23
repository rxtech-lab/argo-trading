//go:build tools
// +build tools

// This file declares dependencies on build tools.
// These imports ensure the tools are tracked in go.mod and not removed by go mod tidy.
package tools

import (
	_ "golang.org/x/mobile/bind"
)
