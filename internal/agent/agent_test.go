package agent

import (
	"context"
	"errors"
	"refactai/internal/analyzer"
	"refactai/internal/executor"
	"refactai/internal/prompt"
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

	analyzerClient, err := analyzer.New(ws)
	if err != nil {
		t.Fatalf("failed to create analyzer: %v", err)
	}

	exec, err := executor.New(ws)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	return New(
		llm,
		prompt.NewBuilder(),
		analyzerClient,
		exec,
		ws,
	)
}

func TestRun(t *testing.T) {
	t.Run("generates and executes action", func(t *testing.T) {
		llm := &fakeLLM{
			responses: []string{
				"Create a simple Go program that prints a message.",
				`package main

			import "fmt"

			func main() {
				fmt.Println("hello from agent")
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

			import "fmt"

			func main() {
				fmt.Println("refactored")
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

	agent := New(llm, prompt.NewBuilder(), analyzerClient, exec, ws)

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

	if !strings.Contains(llm.prompts[0], "function_too_long") {
		t.Fatal("expected plan prompt to contain analyzer finding")
	}

	if !strings.Contains(llm.prompts[0], "main.go") {
		t.Fatal("expected plan prompt to contain analyzed file")
	}
}
