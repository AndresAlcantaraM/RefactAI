package agent

import (
	"context"
	"errors"
	"refactai/internal/analyzer"
	"refactai/internal/executor"
	"refactai/internal/prompt"
	"refactai/internal/validator"
	"refactai/internal/workspace"
	"strings"
	"testing"
)

type fakeLLM struct {
	responses []string
	errors    []error
	calls     int
	prompts   []string
}

func (f *fakeLLM) Generate(ctx context.Context, prompt string) (string, error) {
	call := f.calls
	f.calls++

	if call < len(f.errors) && f.errors[call] != nil {
		return "", f.errors[call]
	}

	if call >= len(f.responses) {
		return "", errors.New("unexpected LLM call")
	}

	f.prompts = append(f.prompts, prompt)

	return f.responses[call], nil
}

func newTestAgent(t *testing.T, llm LLM) *Agent {
	t.Helper()

	ws, err := workspace.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	err = ws.WriteFile("go.mod", []byte(`module testproject
go 1.23
`))
	if err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	err = ws.WriteFile("main.go", []byte(`package main

func main() {}
`))
	if err != nil {
		t.Fatalf("failed to create main.go: %v", err)
	}

	analyzerClient, err := analyzer.New(ws)
	if err != nil {
		t.Fatalf("failed to create analyzer: %v", err)
	}

	exec, err := executor.New(ws)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	validatorClient, err := validator.New(ws)
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	return New(
		llm,
		prompt.NewBuilder(),
		analyzerClient,
		exec,
		ws,
		validatorClient,
	)
}

func TestRun(t *testing.T) {
	t.Run("generates and executes action", func(t *testing.T) {
		llm := &fakeLLM{
			responses: []string{
				"Create a simple Go program that prints a message.",
				`package main

			import (
				"fmt"
				"os"
			)

			func main() {
				fmt.Println("hello from agent")
				if err := os.WriteFile("main.go", []byte("package main\n\nfunc main() {}\n// updated by agent\n"), 0644); err != nil {
					panic(err)
				}
			}`,
			},
		}

		agent := newTestAgent(t, llm)

		result, err := agent.Run(context.Background(), "Create a Go program that prints hello.")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if result.Task == "" {
			t.Fatal("expected task")
		}
		if result.Plan == "" {
			t.Fatal("expected plan")
		}
		if result.Code == "" {
			t.Fatal("expected result")
		}
		if result.Execution.ExitCode != 0 {
			t.Fatalf("expected exit code 0, got %d", result.Execution.ExitCode)
		}
		if result.Execution.Stdout != "hello from agent\n" {
			t.Fatalf("expected %q, got %q", "hello from agent\n", result.Execution.Stdout)
		}

		if result.Validation.Skipped {
			t.Fatalf("expected validation to run")
		}

		if result.Validation.ExitCode != 0 {
			t.Fatalf("expected validation exit code 0, got %d", result.Validation.ExitCode)
		}

		if llm.calls != 2 {
			t.Fatalf("expected 2 LLM calls, got %d", llm.calls)
		}
	})

	t.Run("returns error when plan generation fails", func(t *testing.T) {
		expectedErr := errors.New("llm unavailable")

		llm := &fakeLLM{
			errors: []error{expectedErr},
		}

		agent := newTestAgent(t, llm)

		_, err := agent.Run(context.Background(), "Improve this repository.")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected wrapped LLM error, got %v", err)
		}
	})

	t.Run("rejects empty task", func(t *testing.T) {
		llm := &fakeLLM{}

		agent := newTestAgent(t, llm)

		_, err := agent.Run(context.Background(), "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if llm.calls != 0 {
			t.Fatalf("expected no LLM calls, got %d", llm.calls)
		}
	})
}

func TestRunRetriesAfterValidationFailure(t *testing.T) {
	llm := &fakeLLM{
		responses: []string{
			"Create a simple Go program.",
			`package main

func main() {
	doesNotExist()
}`,
			`package main

import "os"

func main() {
	if err := os.WriteFile("main.go", []byte("package main\n\nfunc main() {}\n// fixed by agent\n"), 0644); err != nil {
		panic(err)
	}
}`,
		},
	}

	agent := newTestAgent(t, llm)

	result, err := agent.Run(
		context.Background(),
		"Create a valid Go program.",
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Execution.ExitCode != 0 {
		t.Fatalf("expected execution exit code 0, got %d", result.Execution.ExitCode)
	}

	if result.Validation.ExitCode != 0 {
		t.Fatalf("expected final validation exit code 0, got %d", result.Validation.ExitCode)
	}

	if llm.calls != 3 {
		t.Fatalf("expected 3 LLM calls, got %d", llm.calls)
	}

	if len(llm.prompts) != 3 {
		t.Fatalf("expected 3 prompts, got %d", len(llm.prompts))
	}

	if !strings.Contains(llm.prompts[2], "VALIDATION EXIT CODE:") {
		t.Fatal("expected corrective prompt to contain validation feedback")
	}

	if !strings.Contains(llm.prompts[2], "previously generated") {
		t.Fatal("expected corrective prompt to contain failure context")
	}
}

func TestRunStopsAfterMaxAttempts(t *testing.T) {
	llm := &fakeLLM{
		responses: []string{
			"Create a Go program.",
			`package main

import "os"

func main() {
	os.Exit(1)
}`,
			`package main

import "os"

func main() {
	os.Exit(2)
}`,
			`package main

import "os"

func main() {
	os.Exit(3)
}`,
		},
	}

	agent := newTestAgent(t, llm)

	result, err := agent.Run(
		context.Background(),
		"Create a failing program.",
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if result.Code == "" {
		t.Fatal("expected generated code in result")
	}

	if result.Execution.ExitCode != 3 {
		t.Fatalf("expected final exit code 3, got %d", result.Execution.ExitCode)
	}

	if llm.calls != 4 {
		t.Fatalf("expected 4 LLM calls, got %d", llm.calls)
	}
}

func TestRunUsesAnalyzerFindings(t *testing.T) {
	ws, err := workspace.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	err = ws.WriteFile("main.go", []byte(`
		package main

		func main() {
			println("hello")
		}

		func tooLong() {
			println("1")
			println("2")
			println("3")
			println("4")
			println("5")
			println("6")
			println("7")
			println("8")
			println("9")
			println("10")
			println("11")
		}
		`))

	if err != nil {
		t.Fatalf("failed to write Go file: %v", err)
	}

	llm := &fakeLLM{
		responses: []string{
			"Refactor the tooLong function.",
			`package main

			import (
				"fmt"
				"os"
			)

			func main() {
				fmt.Println("refactored")
				if err := os.WriteFile("main.go", []byte("package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n"), 0644); err != nil {
					panic(err)
				}
			}`,
		},
	}

	analyzerClient, err := analyzer.New(ws)
	if err != nil {
		t.Fatalf("failed to create analyzer: %v", err)
	}

	exec, err := executor.New(ws)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	validatorClient, err := validator.New(ws)
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	agent := New(llm, prompt.NewBuilder(), analyzerClient, exec, ws, validatorClient)

	result, err := agent.Run(context.Background(), "Improve maintainability of this repository.")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Plan == "" {
		t.Fatalf("expected plan")
	}

	if result.Execution.ExitCode != 0 {
		t.Fatalf("expected execution code 0, got %d", result.Execution.ExitCode)
	}

	if result.Validation.ExitCode != 0 {
		t.Fatalf("expected validation exit code 0, got %d", result.Validation.ExitCode)
	}

	if !strings.Contains(llm.prompts[0], "function_too_long") {
		t.Fatal("expected plan prompt to contain analyzer finding")
	}

	if !strings.Contains(llm.prompts[0], "main.go") {
		t.Fatal("expected plan prompt to contain analyzed file")
	}
}
