package validator

import (
	"context"
	"refactai/internal/workspace"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("rejects nil workspace", func(t *testing.T) {
		_, err := New(nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestValidate(t *testing.T) {
	t.Run("skips workspace without go.mod", func(t *testing.T) {
		ws, err := workspace.New(t.TempDir())
		if err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}

		validator, err := New(ws)
		if err != nil {
			t.Fatalf("failed to create validator: %v", err)
		}

		result, err := validator.Validate(context.Background())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if !result.Skipped {
			t.Fatal("expected validation to be skipped")
		}

		if result.ExitCode != 0 {
			t.Fatalf("expected exit code 0, got %d", result.ExitCode)
		}
	})

	t.Run("passes valid Go project", func(t *testing.T) {
		ws, err := workspace.New(t.TempDir())
		if err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}

		writeGoProject(t, ws, `
		package main

		func main() {}
		`)

		validator, err := New(ws)
		if err != nil {
			t.Fatalf("failed to create validator: %v", err)
		}

		result, err := validator.Validate(context.Background())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if result.Skipped {
			t.Fatal("expected validation to run")
		}

		if result.ExitCode != 0 {
			t.Fatalf("expected exit code 0, got %d", result.ExitCode)
		}
	})

	t.Run("fails invalid Go project", func(t *testing.T) {
		ws, err := workspace.New(t.TempDir())
		if err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}

		writeGoProject(t, ws, `
		package main

		func main() {
			doesNotExist()
		}
		`)

		validator, err := New(ws)
		if err != nil {
			t.Fatalf("failed to create validator: %v", err)
		}

		result, err := validator.Validate(context.Background())
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}

		if result.ExitCode == 0 {
			t.Fatal("expected non-zero exit code")
		}

		if result.Stderr == "" && result.Stdout == "" {
			t.Fatal("expected validation output")
		}
	})

	t.Run("fails when tests fail", func(t *testing.T) {
		ws, err := workspace.New(t.TempDir())
		if err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}

		writeGoProject(t, ws, `
		package main

		func main() {}
		`)

		err = ws.WriteFile("main_test.go", []byte(`
		package main

		import "testing"

		func TestFailure(t *testing.T) {
			t.Fatal("intentional failure")
		}
		`))
		if err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		validator, err := New(ws)
		if err != nil {
			t.Fatalf("failed to create validator: %v", err)
		}

		result, err := validator.Validate(context.Background())
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}

		if result.ExitCode == 0 {
			t.Fatal("expected non-zero exit code")
		}

		output := result.Stdout + result.Stderr

		if !strings.Contains(output, "intentional failure") {
			t.Fatalf("expected failure output, got %q", output)
		}
	})
}

func TestValidateContextCancellation(t *testing.T) {
	ws, err := workspace.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	writeGoProject(t, ws, `
	package main

	func main() {}
	`)

	validator, err := New(ws)
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := validator.Validate(ctx)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if result.ExitCode != -1 {
		t.Fatalf("expected exit code -1, got %d", result.ExitCode)
	}

}

func writeGoProject(t *testing.T, ws *workspace.Workspace, main string) {
	t.Helper()

	err := ws.WriteFile("go.mod", []byte(
		`
		module testproject

		go 1.23
		`,
	))

	if err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	err = ws.WriteFile("main.go", []byte(main))
	if err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
}
