package wasm

import (
	"context"
	"fmt"

	extism "github.com/extism/go-sdk"
)

type Runtime struct {
	plugin *extism.Plugin
}

func NewRuntime(wasmBytes []byte) (*Runtime, error) {
	manifest := extism.Manifest{
		Wasm: []extism.Wasm{
			extism.WasmData{Data: wasmBytes},
		},
		AllowedHosts: []string{},
		AllowedPaths: map[string]string{},
	}

	config := extism.PluginConfig{
		EnableWasi: true,
	}

	plugin, err := extism.NewPlugin(context.Background(), manifest, config, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin: %w", err)
	}

	return &Runtime{plugin: plugin}, nil
}

func (r *Runtime) ExecuteTransform(input []byte) ([]byte, error) {
	_, result, err := r.plugin.Call("TransformWrapper", input)
	if err != nil {
		return nil, fmt.Errorf("transform execution failed: %w", err)
	}

	return result, nil
}

func (r *Runtime) Close() error {
	if r.plugin != nil {
		return r.plugin.Close(context.Background())
	}
	return nil
}
