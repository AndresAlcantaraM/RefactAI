package executor

import (
	"context"
	"os"
	"refactai/internal/action"
	"refactai/internal/workspace"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	ws, err := workspace.New(os.TempDir())
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	executor, err := New(ws)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if executor == nil {
		t.Fatal("expected executor, got nil")
	}
}

func TestRun(t *testing.T) {
	t.Run("executes action successfully", func(t *testing.T) {
		ws, err := workspace.New(t.TempDir())
		if err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}

		executor, err := New(ws)
		if err != nil {
			t.Fatalf("failed to create executor: %v", err)
		}

		act := action.New(`
		package main

		import "fmt"

		func main() {
			fmt.Println("hello")
		}
		`)

		result, err := executor.Run(context.Background(), act)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("expected exit code 0, got %v", result.ExitCode)
		}
		if !strings.Contains(result.Stdout, "hello") {
			t.Fatalf("expected stdout to contain %q, got %q", "hello", result.Stdout)
		}
	})

	t.Run("returns feedback when action fails", func(t *testing.T) {
		ws, err := workspace.New(t.TempDir())
		if err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}

		executor, err := New(ws)
		if err != nil {
			t.Fatalf("failed to create executor: %v", err)
		}

		act := action.New(`
		package main

		import (
			"fmt"
			"os"
		)

		func main() {
			fmt.Fprintln(os.Stderr, "action failed")
			os.Exit(42)
		}
		`)

		result, err := executor.Run(context.Background(), act)
		if err == nil {
			t.Fatal("expected execution error")
		}

		if result.ExitCode != 1 {
			t.Fatalf("expected exit code 1, got %d", result.ExitCode)
		}

		if !strings.Contains(result.Stderr, "action failed") {
			t.Fatalf("expected stderr to contain %q, got %q", "action failed", result.Stderr)
		}
	})

	t.Run("rejects nil action", func(t *testing.T) {
		ws, err := workspace.New(t.TempDir())
		if err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}

		executor, err := New(ws)
		if err != nil {
			t.Fatalf("failed to create executor: %v", err)
		}

		_, err = executor.Run(context.Background(), nil)

		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("rejects empty action code", func(t *testing.T) {
		ws, err := workspace.New(t.TempDir())
		if err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}

		executor, err := New(ws)
		if err != nil {
			t.Fatalf("failed to create executor: %v", err)
		}

		_, err = executor.Run(context.Background(), action.New(""))

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("respects context timeout", func(t *testing.T) {
		ws, err := workspace.New(t.TempDir())
		if err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}

		executor, err := New(ws)
		if err != nil {
			t.Fatalf("failed to create executor :%v", err)
		}

		act := action.New(`
		package main

		import "time"

		func main() {
			time.Sleep(10 * time.Second)
		}
		`)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		_, err = executor.Run(ctx, act)

		if err == nil {
			t.Fatal("expected execution error, got nil")
		}
	})
}
