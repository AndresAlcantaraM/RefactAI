package comparator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompare(t *testing.T) {
	t.Run("returns no changes for identical workspaces", func(t *testing.T) {
		original := t.TempDir()
		workspace := t.TempDir()

		writeTestFile(t, original, "main.go", "package main\n")
		writeTestFile(t, workspace, "main.go", "package main\n")

		comparator := New()

		changes, err := comparator.Compare(original, workspace)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(changes) != 0 {
			t.Fatalf("expected no changes, got %d", len(changes))
		}
	})

	t.Run("detects modified file", func(t *testing.T) {
		original := t.TempDir()
		workspace := t.TempDir()

		writeTestFile(t, original, "main.go", "func hello() {}\n")
		writeTestFile(t, workspace, "main.go", "func greet() {}\n")

		comparator := New()

		changes, err := comparator.Compare(original, workspace)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(changes) != 1 {
			t.Fatalf("expected 1 change, got %d", len(changes))
		}

		change := changes[0]

		if change.Path != "main.go" {
			t.Fatalf("expected main.go, got %s", change.Path)
		}

		if change.Type != Modified {
			t.Fatalf("expected modified, got %s", change.Type)
		}

		if !strings.Contains(change.Diff, "-func hello() {}") {
			t.Fatalf("expected removed line in diff, got:\n%s", change.Diff)
		}

		if !strings.Contains(change.Diff, "+func greet() {}") {
			t.Fatalf("expected added line in diff, got:\n%s", change.Diff)
		}
	})

	t.Run("detects added file", func(t *testing.T) {
		original := t.TempDir()
		workspace := t.TempDir()

		writeTestFile(t, workspace, "new.go", "package main\n")

		comparator := New()

		changes, err := comparator.Compare(original, workspace)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(changes) != 1 {
			t.Fatalf("expected 1 change, got %d", len(changes))
		}

		if changes[0].Type != Added {
			t.Fatalf("expected added, got %s", changes[0].Type)
		}

		if changes[0].Path != "new.go" {
			t.Fatalf("expected new.go, got %s", changes[0].Path)
		}
	})

	t.Run("detects deleted file", func(t *testing.T) {
		original := t.TempDir()
		workspace := t.TempDir()

		writeTestFile(t, original, "old.go", "package main\n")

		comparator := New()

		changes, err := comparator.Compare(original, workspace)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(changes) != 1 {
			t.Fatalf("expected 1 change, got %d", len(changes))
		}

		if changes[0].Type != Deleted {
			t.Fatalf("expected deleted, got %s", changes[0].Type)
		}

		if changes[0].Path != "old.go" {
			t.Fatalf("expected old.go, got %s", changes[0].Path)
		}
	})

	t.Run("handles nested files", func(t *testing.T) {
		original := t.TempDir()
		workspace := t.TempDir()

		writeTestFile(t, original, "internal/foo/foo.go", "package foo\n")
		writeTestFile(t, workspace, "internal/foo/foo.go", "package foo\n\nfunc Bar() {}\n")

		comparator := New()

		changes, err := comparator.Compare(original, workspace)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(changes) != 1 {
			t.Fatalf("expected 1 change, got %d", len(changes))
		}

		if changes[0].Path != "internal/foo/foo.go" {
			t.Fatalf("unexpected path: %s", changes[0].Path)
		}

		if changes[0].Type != Modified {
			t.Fatalf("expected modified, got %s", changes[0].Type)
		}
	})
}

func writeTestFile(t *testing.T, root, path, content string) {
	t.Helper()

	fullPath := filepath.Join(root, filepath.FromSlash(path))

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
}

func TestApply(t *testing.T) {
	originalRoot := t.TempDir()
	workspaceRoot := t.TempDir()

	if err := os.WriteFile(
		filepath.Join(originalRoot, "modified.txt"),
		[]byte("original content"),
		0644,
	); err != nil {
		t.Fatalf("failed to create original modified.txt: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(originalRoot, "deleted.txt"),
		[]byte("this file will be deleted"),
		0644,
	); err != nil {
		t.Fatalf("failed to create original deleted.txt: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(workspaceRoot, "modified.txt"),
		[]byte("modified content"),
		0644,
	); err != nil {
		t.Fatalf("failed to create workspace modified.txt: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(workspaceRoot, "added.txt"),
		[]byte("new file"),
		0644,
	); err != nil {
		t.Fatalf("failed to create workspace added.txt: %v", err)
	}

	comparator := New()

	changes := []Change{
		{
			Path: "modified.txt",
			Type: Modified,
		},
		{
			Path: "added.txt",
			Type: Added,
		},
		{
			Path: "deleted.txt",
			Type: Deleted,
		},
	}

	if err := comparator.Apply(originalRoot, workspaceRoot, changes); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	content, err := os.ReadFile(filepath.Join(originalRoot, "modified.txt"))
	if err != nil {
		t.Fatalf("failed to read modified.txt: %v", err)
	}

	if string(content) != "modified content" {
		t.Fatalf("expected modified content, got %q", string(content))
	}

	content, err = os.ReadFile(filepath.Join(originalRoot, "added.txt"))
	if err != nil {
		t.Fatalf("failed to read added.txt: %v", err)
	}

	if string(content) != "new file" {
		t.Fatalf("expected new file content, got %q", string(content))
	}

	if _, err := os.Stat(filepath.Join(originalRoot, "deleted.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected deleted.txt to be removed")
	}
}
