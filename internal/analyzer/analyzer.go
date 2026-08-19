package analyzer

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"refactai/internal/workspace"
	"strings"
)

const (
	maxFunctionLines = 10
	maxParameters    = 5
)

type Analyzer struct {
	workspace *workspace.Workspace
}

type Finding struct {
	File    string
	Line    int
	Type    string
	Message string
}

func New(ws *workspace.Workspace) (*Analyzer, error) {
	if ws == nil {
		return nil, fmt.Errorf("workspace cannot be nil")
	}
	return &Analyzer{
		workspace: ws,
	}, nil
}

func (a *Analyzer) Analyze() ([]Finding, error) {
	files, err := a.workspace.ListFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace files: %w", err)
	}

	var findings []Finding

	for _, file := range files {
		if !strings.HasSuffix(file, ".go") {
			continue
		}
		fileFindings, err := a.analyzeFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze %s: %w", file, err)
		}

		findings = append(findings, fileFindings...)
	}

	return findings, nil
}

func (a *Analyzer) analyzeFile(path string) ([]Finding, error) {
	content, err := a.workspace.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file: %w", err)
	}

	var findings []Finding

	ast.Inspect(file, func(node ast.Node) bool {
		function, ok := node.(*ast.FuncDecl)
		if !ok {
			return true
		}

		findings = append(findings, a.checkFunctionLenght(path, fset, function))

		findings = append(findings, a.checkParameters(path, fset, function))

		return true
	})

	return a.filterEmptyFindings(findings), nil
}

func (a *Analyzer) checkFunctionLenght(path string, fset *token.FileSet, function *ast.FuncDecl) Finding {
	start := fset.Position(function.Pos())
	end := fset.Position(function.End())

	lines := end.Line - start.Line + 1

	if lines <= maxFunctionLines {
		return Finding{}
	}

	return Finding{
		File:    filepath.ToSlash(path),
		Line:    start.Line,
		Type:    "function_too_long",
		Message: fmt.Sprintf("function %s has %d lines", function.Name.Name, lines),
	}
}

func (a *Analyzer) checkParameters(path string, fset *token.FileSet, function *ast.FuncDecl) Finding {
	if function.Type.Params == nil {
		return Finding{}
	}

	parameterCount := 0

	for _, field := range function.Type.Params.List {
		if len(field.Names) == 0 {
			parameterCount++
			continue
		}
		parameterCount += len(field.Names)
	}

	if parameterCount <= maxParameters {
		return Finding{}
	}

	position := fset.Position(function.Pos())

	return Finding{
		File:    filepath.ToSlash(path),
		Line:    position.Line,
		Type:    "too_many_parameters",
		Message: fmt.Sprintf("function %s has %d parameters", function.Name.Name, parameterCount),
	}
}

func (a *Analyzer) filterEmptyFindings(findings []Finding) []Finding {
	result := make([]Finding, 0, len(findings))

	for _, finding := range findings {
		if finding.Type == "" {
			continue
		}

		result = append(result, finding)
	}

	return result
}
