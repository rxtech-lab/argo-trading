//go:build wasip1

package strategy

import (
	wasm "github.com/knqyf263/go-plugin/wasm"

	"github.com/rxtech-lab/argo-trading/internal/version"
)

//go:wasmexport trading_strategy_engine_version
func _trading_strategy_engine_version() uint64 {
	// Use the version from internal/version which is set when argo-trading is built/released
	ptr, size := wasm.ByteToPtr([]byte(version.GetVersion()))
	return (uint64(ptr) << uint64(32)) | uint64(size)
}
