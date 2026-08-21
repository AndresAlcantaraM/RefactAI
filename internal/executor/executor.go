package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"refactai/internal/action"
	"refactai/internal/workspace"
	"runtime"
	"strings"

	"golang.org/x/tools/imports"
)

const (
	actionFileName   = "action.go"
	toolsFileName    = "refactai_tools.go"
	actionBinaryName = "action"
)

const toolsSource = `package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

// replaceExact replaces oldStr with newStr in the file at path.
// It fails loudly if oldStr is not found, or found more than once,
// instead of silently doing nothing or guessing.
func replaceExact(path, oldStr, newStr string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("replaceExact: cannot read %s: %w", path, err)
	}
	source := string(content)
	count := strings.Count(source, oldStr)
	if count == 0 {
		return fmt.Errorf("replaceExact: old text not found in %s (must match exactly, including whitespace)", path)
	}
	if count > 1 {
		return fmt.Errorf("replaceExact: old text found %d times in %s, add more surrounding context to make it unique", count, path)
	}
	updated := strings.Replace(source, oldStr, newStr, 1)
	if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
		return fmt.Errorf("replaceExact: cannot write %s: %w", path, err)
	}
	return nil
}

// createFile creates a brand new file. It fails if the file already
// exists, so it cannot be used to silently overwrite an existing file.
func createFile(path, content string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("createFile: %s already exists, use replaceExact to modify it", path)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("createFile: cannot write %s: %w", path, err)
	}
	return nil
}

// deleteFile removes an existing file.
func deleteFile(path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("deleteFile: %s does not exist", path)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("deleteFile: cannot remove %s: %w", path, err)
	}
	return nil
}

// replaceFunction finds a top-level function or method by name using
// Go's AST (not text matching) and replaces its entire declaration
// with newSource. newSource may contain more than one declaration
// (e.g. the modified function plus new helper functions); all of
// them are spliced in at the original function's position. The
// result is gofmt'd and syntax-checked before writing.
func replaceFunction(path, functionName, newSource string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("replaceFunction: cannot read %s: %w", path, err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("replaceFunction: cannot parse %s: %w", path, err)
	}
	var target *ast.FuncDecl
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == functionName {
			target = fn
			break
		}
	}
	if target == nil {
		return fmt.Errorf("replaceFunction: function %q not found in %s", functionName, path)
	}
	start := fset.Position(target.Pos()).Offset
	end := fset.Position(target.End()).Offset
	var buf bytes.Buffer
	buf.Write(content[:start])
	buf.WriteString(strings.TrimSpace(newSource))
	buf.WriteString("\n")
	buf.Write(content[end:])
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("replaceFunction: result is not valid Go source: %w", err)
	}
	if err := os.WriteFile(path, formatted, 0644); err != nil {
		return fmt.Errorf("replaceFunction: cannot write %s: %w", path, err)
	}
	return nil
}
`

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type Executor struct {
	workspace *workspace.Workspace
}

func New(workspace *workspace.Workspace) (*Executor, error) {
	if workspace == nil {
		return nil, fmt.Errorf("workspace cannot be nil")
	}
	return &Executor{workspace: workspace}, nil
}

func (e *Executor) Run(ctx context.Context, act *action.Action) (Result, error) {
	if act == nil {
		return Result{}, fmt.Errorf("action cannot be nil")
	}
	if act.Code == "" {
		return Result{}, fmt.Errorf("action code cannot be empty")
	}
	if err := e.workspace.WriteFile(actionFileName, []byte(act.Code)); err != nil {
		return Result{}, fmt.Errorf("failed to write action: %w", err)
	}
	if err := e.workspace.WriteFile(toolsFileName, []byte(toolsSource)); err != nil {
		return Result{}, fmt.Errorf("failed to write tools file: %w", err)
	}

	binaryName := actionBinaryName
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(e.workspace.Root(), binaryName)

	defer func() { _ = e.workspace.DeleteFile(actionFileName) }()
	defer func() { _ = e.workspace.DeleteFile(toolsFileName) }()
	defer func() { _ = e.workspace.DeleteFile(binaryName) }()

	if err := e.build(ctx, binaryName); err != nil {
		return Result{Stderr: err.Error(), ExitCode: -1}, fmt.Errorf("failed to build action: %w", err)
	}
	result, err := e.execute(ctx, binaryPath)
	if err != nil {
		return result, err
	}

	if fixErr := e.fixImports(); fixErr != nil {
		return result, fmt.Errorf("failed to fix imports after action execution: %w", fixErr)
	}

	return result, nil
}

func (e *Executor) build(ctx context.Context, binaryName string) error {
	cmd := exec.CommandContext(ctx, "go", "build", "-o", binaryName, actionFileName, toolsFileName)
	cmd.Dir = e.workspace.Root()
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) > 0 {
			return fmt.Errorf("%s: %w", output, err)
		}
		return err
	}
	return nil
}

func (e *Executor) execute(ctx context.Context, binaryPath string) (Result, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Dir = e.workspace.Root()
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: 0}
	if err == nil {
		return result, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
	} else {
		result.ExitCode = -1
	}
	return result, fmt.Errorf("failed to execute action: %w", err)
}

func (e *Executor) fixImports() error {
	files, err := e.workspace.ListFiles()
	if err != nil {
		return fmt.Errorf("failed to list workspace files: %w", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file, ".go") {
			continue
		}
		if file == actionFileName || file == toolsFileName {
			continue
		}

		content, err := e.workspace.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		fullPath := filepath.Join(e.workspace.Root(), file)
		formatted, err := imports.Process(fullPath, content, nil)
		if err != nil {
			continue
		}

		if bytes.Equal(formatted, content) {
			continue
		}

		if err := e.workspace.WriteFile(file, formatted); err != nil {
			return fmt.Errorf("failed to write formatted %s: %w", file, err)
		}
	}

	return nil
}
