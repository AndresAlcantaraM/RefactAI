package executor

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"refactai/internal/action"
	"refactai/internal/workspace"
)

func newTestExecutor(t *testing.T) (*Executor, *workspace.Workspace) {
	t.Helper()

	ws, err := workspace.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	exec, err := New(ws)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	return exec, ws
}

func TestNew(t *testing.T) {
	ws, err := workspace.New(t.TempDir())
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
	t.Run("executes action successfully and cleans temporary files", func(t *testing.T) {
		exec, ws := newTestExecutor(t)

		code := `package main

		import "fmt"

		func main() {
			fmt.Println("hello executor")
		}`

		result, err := exec.Run(context.Background(), action.New(code))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if result.ExitCode != 0 {
			t.Fatalf("expected exit code 0, got %d", result.ExitCode)
		}

		if !strings.Contains(result.Stdout, "hello executor") {
			t.Fatalf("expected stdout to contain %q, got %q", "hello executor", result.Stdout)
		}

		assertTemporaryFilesRemoved(t, ws)
	})

	// Agregado del Bloque 1: Prueba de fallo en tiempo de ejecución (exit code custom + stderr)
	t.Run("returns feedback when action fails at runtime", func(t *testing.T) {
		exec, ws := newTestExecutor(t)

		code := `package main

		import (
			"fmt"
			"os"
		)

		func main() {
			fmt.Fprintln(os.Stderr, "action failed")
			os.Exit(42)
		}`

		result, err := exec.Run(context.Background(), action.New(code))
		if err == nil {
			t.Fatal("expected execution error, got nil")
		}

		if result.ExitCode != 42 {
			t.Fatalf("expected exit code 42, got %d", result.ExitCode)
		}

		if !strings.Contains(result.Stderr, "action failed") {
			t.Fatalf("expected stderr to contain %q, got %q", "action failed", result.Stderr)
		}

		assertTemporaryFilesRemoved(t, ws)
	})

	t.Run("returns build error and cleans action file", func(t *testing.T) {
		exec, ws := newTestExecutor(t)

		code := `package main

		func main() {
			thisDoesNotCompile()
		}`

		result, err := exec.Run(context.Background(), action.New(code))
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if result.ExitCode != -1 {
			t.Fatalf("expected exit code -1, got %d", result.ExitCode)
		}

		assertTemporaryFilesRemoved(t, ws)
	})

	t.Run("rejects nil action", func(t *testing.T) {
		exec, _ := newTestExecutor(t)

		_, err := exec.Run(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("rejects empty action", func(t *testing.T) {
		exec, _ := newTestExecutor(t)

		_, err := exec.Run(context.Background(), action.New(""))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRunContextCancellation(t *testing.T) {
	exec, ws := newTestExecutor(t)

	code := `package main

	import "time"

	func main() {
		time.Sleep(10 * time.Second)
	}`

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := exec.Run(ctx, action.New(code))
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if result.ExitCode != -1 {
		t.Fatalf("expected exit code -1, got %d", result.ExitCode)
	}

	assertTemporaryFilesRemoved(t, ws)
}

func assertTemporaryFilesRemoved(t *testing.T, ws *workspace.Workspace) {
	t.Helper()

	actionPath := filepath.Join(ws.Root(), actionFileName)
	if _, err := os.Stat(actionPath); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be removed, got err=%v", actionFileName, err)
	}

	binaryName := actionBinaryName
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	binaryPath := filepath.Join(ws.Root(), binaryName)
	if _, err := os.Stat(binaryPath); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be removed, got err=%v", binaryName, err)
	}
}
