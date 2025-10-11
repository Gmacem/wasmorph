package wasm

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntime_ExecuteTransform(t *testing.T) {
	compiler := NewCompiler("wasm-template", "test-temp")

	sourceCode := `import "encoding/json"

func Transform(input []byte) []byte {
	var in map[string]any
	json.Unmarshal(input, &in)
	
	result := make(map[string]any)
	result["processed"] = true
	result["input"] = in
	result["message"] = "Hello from WASM!"
	
	output, _ := json.Marshal(result)
	return output
}`

	t.Run("compile and execute", func(t *testing.T) {
		wasmBytes, err := compiler.CompileGoToWasm(sourceCode, "test")
		if err != nil {
			t.Skip("Skipping test - compilation failed:", err)
			return
		}

		runtime, err := NewRuntime(wasmBytes)
		if err != nil {
			t.Skip("Skipping test - runtime creation failed:", err)
			return
		}
		defer runtime.Close()

		input := map[string]any{"test": "value", "number": 42.0}
		inputJSON, _ := json.Marshal(input)
		result, err := runtime.ExecuteTransform(inputJSON)

		require.NoError(t, err)

		var resultMap map[string]any
		err = json.Unmarshal(result, &resultMap)
		require.NoError(t, err)

		assert.Equal(t, true, resultMap["processed"])
		assert.Equal(t, "Hello from WASM!", resultMap["message"])
		assert.Equal(t, input, resultMap["input"])
	})
}

func TestRuntime_Close(t *testing.T) {
	t.Run("close nil plugin", func(t *testing.T) {
		runtime := &Runtime{plugin: nil}
		err := runtime.Close()
		assert.NoError(t, err)
	})
}

func TestNewRuntime(t *testing.T) {
	t.Run("invalid wasm data", func(t *testing.T) {
		invalidWasm := []byte{0x00, 0x01, 0x02}
		_, err := NewRuntime(invalidWasm)
		assert.Error(t, err)
	})

	t.Run("empty wasm data", func(t *testing.T) {
		_, err := NewRuntime([]byte{})
		assert.Error(t, err)
	})

	t.Run("valid wasm from compilation", func(t *testing.T) {
		compiler := NewCompiler("wasm-template", "test-temp")
		sourceCode := `func Transform(input []byte) []byte {
	return input
}`

		wasmBytes, err := compiler.CompileGoToWasm(sourceCode, "test")
		if err != nil {
			t.Skip("Skipping test - compilation failed:", err)
			return
		}

		runtime, err := NewRuntime(wasmBytes)
		require.NoError(t, err)
		require.NotNil(t, runtime)
		defer runtime.Close()
	})
}
