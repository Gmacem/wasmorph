package wasm

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const wasmTemplate = `package main

import (
	"github.com/extism/go-pdk"
)

// User's transform function will be inserted here
%s

//go:export TransformWrapper
func TransformWrapper() int32 {
	inputBytes := pdk.Input()
	result := Transform(inputBytes)
	pdk.Output(result)
	return 0
}

func main() {
	select {}
}
`

type Compiler struct {
	templateDir string
	tempBaseDir string
}

func NewCompiler(templateDir, tempBaseDir string) *Compiler {
	return &Compiler{
		templateDir: templateDir,
		tempBaseDir: tempBaseDir,
	}
}

func (c *Compiler) CompileGoToWasm(sourceCode, ruleName string) ([]byte, error) {
	if err := c.validateGoCode(sourceCode); err != nil {
		return nil, fmt.Errorf("code validation failed: %w", err)
	}

	tempDir, err := c.createTempDir()
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := c.copyTemplate(tempDir); err != nil {
		return nil, fmt.Errorf("failed to copy template: %w", err)
	}

	if err := c.replaceTransformFunction(tempDir, sourceCode); err != nil {
		return nil, fmt.Errorf("failed to replace transform function: %w", err)
	}

	wasmFile := filepath.Join(tempDir, "main.wasm")
	if err := c.compileWithTinyGo(tempDir, wasmFile); err != nil {
		return nil, fmt.Errorf("compilation failed: %w", err)
	}

	wasmBytes, err := os.ReadFile(wasmFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read wasm file: %w", err)
	}

	return wasmBytes, nil
}

func (c *Compiler) createTempDir() (string, error) {
	hash := make([]byte, 16)
	if _, err := rand.Read(hash); err != nil {
		return "", err
	}
	hashStr := hex.EncodeToString(hash)

	tempDir := filepath.Join(c.tempBaseDir, "wasmorph-build-"+hashStr)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", err
	}

	return tempDir, nil
}

func (c *Compiler) copyTemplate(tempDir string) error {
	return c.copyDir(c.templateDir, tempDir)
}

func (c *Compiler) copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := c.copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := c.copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Compiler) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func (c *Compiler) replaceTransformFunction(tempDir, sourceCode string) error {
	mainGoPath := filepath.Join(tempDir, "main.go")
	wasmSourceCode := fmt.Sprintf(wasmTemplate, sourceCode)
	return os.WriteFile(mainGoPath, []byte(wasmSourceCode), 0644)
}

func (c *Compiler) compileWithTinyGo(tempDir, wasmFile string) error {
	mainGoPath := filepath.Join(tempDir, "main.go")

	absMainGoPath, err := filepath.Abs(mainGoPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	absWasmFile, err := filepath.Abs(wasmFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	cmd := exec.Command("tinygo", "build", "-o", absWasmFile, "-target", "wasi", absMainGoPath)
	cmd.Dir = tempDir

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Args[0], cmd.Args[1:]...)
	cmd.Dir = tempDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tinygo compilation failed: %s", stderr.String())
	}

	return nil
}

func (c *Compiler) validateGoCode(sourceCode string) error {
	tempFile := `package main

` + sourceCode

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "main.go", tempFile, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("invalid Go syntax: %w", err)
	}

	hasTransform := false
	var validationErr error
	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			if fn.Name.Name == "Transform" {
				hasTransform = true
				if err := c.validateTransformSignature(fn); err != nil {
					validationErr = err
					return false
				}
			}
		}
		return true
	})

	if !hasTransform {
		return fmt.Errorf("transform function not found")
	}

	if validationErr != nil {
		return validationErr
	}

	return nil
}

func (c *Compiler) validateTransformSignature(fn *ast.FuncDecl) error {
	if fn.Type.Params.NumFields() != 1 {
		return fmt.Errorf("transform must have exactly 1 parameter")
	}

	if fn.Type.Results.NumFields() != 1 {
		return fmt.Errorf("transform must return exactly 1 value")
	}

	param := fn.Type.Params.List[0]
	if len(param.Names) != 1 || !c.isByteSlice(param.Type) {
		return fmt.Errorf("parameter must be []byte")
	}

	result := fn.Type.Results.List[0]
	if !c.isByteSlice(result.Type) {
		return fmt.Errorf("transform must return []byte")
	}

	return nil
}

func (c *Compiler) isByteSlice(expr ast.Expr) bool {
	if arrayType, ok := expr.(*ast.ArrayType); ok {
		if ident, ok := arrayType.Elt.(*ast.Ident); ok && ident.Name == "byte" {
			return true
		}
	}
	return false
}
