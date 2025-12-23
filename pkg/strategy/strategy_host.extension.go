//go:build !wasip1

package strategy

import (
	context "context"
	errors "errors"
	fmt "fmt"

	sys "github.com/tetratelabs/wazero/sys"
)

func (p *TradingStrategyPlugin) LoadFromBytes(ctx context.Context, bytes []byte, hostFunctions StrategyApi) (tradingStrategy, error) {
	// Create a new runtime so that multiple modules will not conflict
	r, err := p.newRuntime(ctx)
	if err != nil {
		return nil, err
	}

	h := _strategyApi{hostFunctions}

	if err := h.Instantiate(ctx, r); err != nil {
		return nil, err
	}

	// Compile the WebAssembly module using the default configuration.
	code, err := r.CompileModule(ctx, bytes)
	if err != nil {
		return nil, err
	}

	// InstantiateModule runs the "_start" function, WASI's "main".
	module, err := r.InstantiateModule(ctx, code, p.moduleConfig)
	if err != nil {
		// Note: Most compilers do not exit the module after running "_start",
		// unless there was an Error. This allows you to call exported functions.
		if exitErr, ok := err.(*sys.ExitError); ok && exitErr.ExitCode() != 0 {
			return nil, fmt.Errorf("unexpected exit_code: %d", exitErr.ExitCode())
		} else if !ok {
			return nil, err
		}
	}

	// Compare API versions with the loading plugin
	apiVersion := module.ExportedFunction("trading_strategy_api_version")
	if apiVersion == nil {
		return nil, errors.New("trading_strategy_api_version is not exported")
	}
	results, err := apiVersion.Call(ctx)
	if err != nil {
		return nil, err
	} else if len(results) != 1 {
		return nil, errors.New("invalid trading_strategy_api_version signature")
	}
	if results[0] != TradingStrategyPluginAPIVersion {
		return nil, fmt.Errorf("API version mismatch, host: %d, plugin: %d", TradingStrategyPluginAPIVersion, results[0])
	}

	initialize := module.ExportedFunction("trading_strategy_initialize")
	if initialize == nil {
		return nil, errors.New("trading_strategy_initialize is not exported")
	}
	processdata := module.ExportedFunction("trading_strategy_process_data")
	if processdata == nil {
		return nil, errors.New("trading_strategy_process_data is not exported")
	}
	name := module.ExportedFunction("trading_strategy_name")
	if name == nil {
		return nil, errors.New("trading_strategy_name is not exported")
	}
	getconfigschema := module.ExportedFunction("trading_strategy_get_config_schema")
	if getconfigschema == nil {
		return nil, errors.New("trading_strategy_get_config_schema is not exported")
	}
	getdescription := module.ExportedFunction("trading_strategy_get_description")
	if getdescription == nil {
		return nil, errors.New("trading_strategy_get_description is not exported")
	}

	malloc := module.ExportedFunction("malloc")
	if malloc == nil {
		return nil, errors.New("malloc is not exported")
	}

	free := module.ExportedFunction("free")
	if free == nil {
		return nil, errors.New("free is not exported")
	}
	return &tradingStrategyPlugin{
		runtime:         r,
		module:          module,
		malloc:          malloc,
		free:            free,
		initialize:      initialize,
		processdata:     processdata,
		name:            name,
		getconfigschema: getconfigschema,
		getdescription:  getdescription,
	}, nil
}
