package e2e

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"refactai/internal/agent"
	"refactai/internal/analyzer"
	"refactai/internal/comparator"
	"refactai/internal/executor"
	"refactai/internal/prompt"
	"refactai/internal/validator"
	"refactai/internal/workspace"
)

type fakeLLM struct {
	responses []string
	calls     int
}

func (f *fakeLLM) Generate(ctx context.Context, prompt string) (string, error) {
	if f.calls >= len(f.responses) {
		return "", os.ErrNotExist
	}

	response := f.responses[f.calls]
	f.calls++

	return response, nil
}

func TestAgentWorkflowEndToEnd(t *testing.T) {
	// ------------------------------------------------------------
	// 1. Create original project.
	// ------------------------------------------------------------
	projectRoot := t.TempDir()

	mainGo := `package main

import "fmt"

func hello() {
	fmt.Println("hello")
}

func main() {
	hello()
}
`

	if err := os.WriteFile(
		filepath.Join(projectRoot, "main.go"),
		[]byte(mainGo),
		0644,
	); err != nil {
		t.Fatalf("failed to create main.go: %v", err)
	}

	goMod := `module demo

go 1.23
`

	if err := os.WriteFile(
		filepath.Join(projectRoot, "go.mod"),
		[]byte(goMod),
		0644,
	); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// ------------------------------------------------------------
	// 2. Create isolated workspace using production code.
	// ------------------------------------------------------------
	workspaceRoot := t.TempDir()

	if err := workspace.CopyDir(projectRoot, workspaceRoot); err != nil {
		t.Fatalf("failed to copy project to workspace: %v", err)
	}

	ws, err := workspace.New(workspaceRoot)
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// ------------------------------------------------------------
	// 3. Build agent dependencies.
	// ------------------------------------------------------------
	analyzerClient, err := analyzer.New(ws)
	if err != nil {
		t.Fatalf("failed to create analyzer: %v", err)
	}

	executorClient, err := executor.New(ws)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	validatorClient, err := validator.New(ws)
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	llm := &fakeLLM{
		responses: []string{
			"Rename hello to greet and update its usage.",
			`package main

		import (
			"fmt"
			"os"
			"strings"
		)

		func main() {
			content, err := os.ReadFile("main.go")
			if err != nil {
				panic(err)
			}

			updated := strings.ReplaceAll(
				string(content),
				"func hello()",
				"func greet()",
			)

			updated = strings.ReplaceAll(
				updated,
				"hello()",
				"greet()",
			)

			if err := os.WriteFile("main.go", []byte(updated), 0644); err != nil {
				panic(err)
			}

			fmt.Println("rename completed")
		}
		`,
		},
	}

	refactAgent := agent.New(
		llm,
		prompt.NewBuilder(),
		analyzerClient,
		executorClient,
		ws,
		validatorClient,
	)

	// ------------------------------------------------------------
	// 4. Run agent.
	// ------------------------------------------------------------
	result, err := refactAgent.Run(
		context.Background(),
		"Rename the hello function to greet and update its usage.",
	)

	if err != nil {
		t.Fatalf("agent run failed: %v", err)
	}

	if result.Execution.ExitCode != 0 {
		t.Fatalf(
			"expected execution exit code 0, got %d",
			result.Execution.ExitCode,
		)
	}

	if result.Validation.ExitCode != 0 {
		t.Fatalf(
			"expected validation exit code 0, got %d",
			result.Validation.ExitCode,
		)
	}

	// ------------------------------------------------------------
	// 5. Generate diff.
	// ------------------------------------------------------------
	comparatorClient := comparator.New()

	changes, err := comparatorClient.Compare(
		projectRoot,
		workspaceRoot,
	)
	if err != nil {
		t.Fatalf("failed to compare projects: %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	change := changes[0]

	if change.Path != "main.go" {
		t.Fatalf(
			"expected change in main.go, got %s",
			change.Path,
		)
	}

	if change.Type != comparator.Modified {
		t.Fatalf(
			"expected modified change, got %s",
			change.Type,
		)
	}

	if !strings.Contains(change.Diff, "-func hello()") {
		t.Fatal("expected diff to contain removed hello function")
	}

	if !strings.Contains(change.Diff, "+func greet()") {
		t.Fatal("expected diff to contain added greet function")
	}

	// ------------------------------------------------------------
	// 6. Apply changes.
	// ------------------------------------------------------------
	if err := comparatorClient.Apply(
		projectRoot,
		workspaceRoot,
		changes,
	); err != nil {
		t.Fatalf("failed to apply changes: %v", err)
	}

	// ------------------------------------------------------------
	// 7. Verify original project was modified.
	// ------------------------------------------------------------
	finalContent, err := os.ReadFile(
		filepath.Join(projectRoot, "main.go"),
	)
	if err != nil {
		t.Fatalf("failed to read final main.go: %v", err)
	}

	finalSource := string(finalContent)

	if !strings.Contains(finalSource, "func greet()") {
		t.Fatal("expected original project to contain func greet()")
	}

	if strings.Contains(finalSource, "func hello()") {
		t.Fatal("expected original project to no longer contain func hello()")
	}

	if !strings.Contains(finalSource, "greet()") {
		t.Fatal("expected original project to contain greet() call")
	}

	// ------------------------------------------------------------
	// 8. Verify LLM interaction.
	// ------------------------------------------------------------
	if llm.calls != 2 {
		t.Fatalf("expected 2 LLM calls, got %d", llm.calls)
	}
}
