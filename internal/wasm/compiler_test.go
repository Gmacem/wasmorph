package wasm

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompiler_ValidateGoCode(t *testing.T) {
	compiler := NewCompiler("wasm-template", "test-temp")

	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name: "valid transform function",
			code: `import "encoding/json"

func Transform(input []byte) []byte {
	var in map[string]any
	json.Unmarshal(input, &in)
	
	result := make(map[string]any)
	result["processed"] = true
	result["input"] = in
	
	output, _ := json.Marshal(result)
	return output
}`,
			wantErr: false,
		},
		{
			name: "missing transform function",
			code: `package main

func main() {
	println("hello")
}`,
			wantErr: true,
		},
		{
			name: "wrong transform signature - no parameters",
			code: `package main

func Transform() map[string]any {
	return make(map[string]any)
}`,
			wantErr: true,
		},
		{
			name: "wrong transform signature - wrong parameter name",
			code: `package main

func Transform(data map[string]any) map[string]any {
	return make(map[string]any)
}`,
			wantErr: true,
		},
		{
			name: "wrong transform signature - wrong return type",
			code: `package main

func Transform(in map[string]any) string {
	return "hello"
}`,
			wantErr: true,
		},
		{
			name: "invalid go syntax",
			code: `package main

func Transform(in map[string]any) map[string]any {
	return make(map[string]any)
	// missing closing brace
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := compiler.validateGoCode(tt.code)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCompiler_ValidateTransformSignature(t *testing.T) {
	compiler := NewCompiler("wasm-template", "test-temp")

	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name: "correct signature",
			code: `func Transform(input []byte) []byte {
	return make([]byte, 0)
}`,
			wantErr: false,
		},
		{
			name: "different parameter name",
			code: `func Transform(data []byte) []byte {
	return make([]byte, 0)
}`,
			wantErr: false,
		},
		{
			name: "multiple parameters",
			code: `func Transform(in []byte, extra string) []byte {
	return make([]byte, 0)
}`,
			wantErr: true,
		},
		{
			name: "no parameters",
			code: `func Transform() []byte {
	return make([]byte, 0)
}`,
			wantErr: true,
		},
		{
			name: "wrong return type",
			code: `func Transform(in []byte) string {
	return "hello"
}`,
			wantErr: true,
		},
		{
			name: "multiple return values",
			code: `func Transform(in []byte) ([]byte, error) {
	return make([]byte, 0), nil
}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the function declaration
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "test.go", "package main\n"+tt.code, parser.ParseComments)
			require.NoError(t, err)

			var fn *ast.FuncDecl
			ast.Inspect(node, func(n ast.Node) bool {
				if f, ok := n.(*ast.FuncDecl); ok && f.Name.Name == "Transform" {
					fn = f
					return false
				}
				return true
			})

			require.NotNil(t, fn, "Transform function not found")

			err = compiler.validateTransformSignature(fn)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCompiler_IsByteSlice(t *testing.T) {
	compiler := NewCompiler("wasm-template", "test-temp")

	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "[]byte",
			code:     "[]byte",
			expected: true,
		},
		{
			name:     "[]string",
			code:     "[]string",
			expected: false,
		},
		{
			name:     "map[string]any",
			code:     "map[string]any",
			expected: false,
		},
		{
			name:     "string",
			code:     "string",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := parser.ParseExpr(tt.code)
			require.NoError(t, err)

			result := compiler.isByteSlice(node)
			assert.Equal(t, tt.expected, result)
		})
	}
}
