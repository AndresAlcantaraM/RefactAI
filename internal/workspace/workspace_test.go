package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("creates workspace directory", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "workspace")

		ws, err := New(root)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if ws == nil {
			t.Fatal("expected workspace, got nil")
		}

		info, err := os.Stat(root)
		if err != nil {
			t.Fatalf("expected workspace directory to exist: %v", err)
		}

		if !info.IsDir() {
			t.Fatal("expected workspace path to be a directory")
		}
	})

	t.Run("rejects empty root", func(t *testing.T) {
		_, err := New("")

		if err == nil {
			t.Fatal("expected error for empty root")
		}
	})
}

func TestWriteFile(t *testing.T) {
	root := t.TempDir()

	ws, err := New(root)
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	t.Run("writes file", func(t *testing.T) {
		content := []byte("package main")

		err := ws.WriteFile("main.go", content)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		filePath := filepath.Join(root, "main.go")

		actual, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read written file :%v", err)
		}

		if string(actual) != string(content) {
			t.Fatalf("expected %q, got %q", string(content), string(actual))
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		content := []byte("package utils")

		err := ws.WriteFile("internal/utils/utils.go", content)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		filePath := filepath.Join(root, "internal", "utils", "utils.go")

		if _, err := os.Stat(filePath); err != nil {
			t.Fatalf("expected file to exist: %v", err)
		}
	})

	t.Run("rejects path outside workspace", func(t *testing.T) {
		err := ws.WriteFile("../virus.txt", []byte("malicious code"))

		if err == nil {
			t.Fatalf("expected error for path outside workspace")
		}
	})
}

func TestReadFile(t *testing.T) {
	ws, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	expected := []byte("hello workspace")

	err = ws.WriteFile("test.txt", expected)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	t.Run("reads file", func(t *testing.T) {
		actual, err := ws.ReadFile("test.txt")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if string(actual) != string(expected) {
			t.Fatalf("expected %q, got %q", string(expected), string(actual))
		}
	})

	t.Run("returns error for nonexistent file", func(t *testing.T) {
		_, err := ws.ReadFile("does-not-exist.txt")

		if err == nil {
			t.Fatal("expected error for nonexistent file")
		}
	})
}

func TestListFiles(t *testing.T) {
	ws, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	files := map[string]string{
		"main.go":                 "package main",
		"internal/utils/utils.go": "package utils",
		"README.md":               "# RefactAI",
	}

	for path, content := range files {
		if err := ws.WriteFile(path, []byte(content)); err != nil {
			t.Fatalf("failed to create %s: %v", path, err)
		}
	}

	result, err := ws.ListFiles()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result) != len(files) {
		t.Fatalf("expected %d files, got %d", len(files), len(result))
	}

	for path := range files {
		found := false

		for _, actual := range result {
			if actual == path {
				t.Logf("Found %v, equals %v", actual, path)
				found = true
				break
			}
		}

		if !found {
			t.Errorf("expected file %q to be listed", path)
		}
	}
}

func TestDeleteFile(t *testing.T) {
	ws, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	err = ws.WriteFile("delete-me.txt", []byte("test"))
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	t.Run("deletes file", func(t *testing.T) {
		err := ws.DeleteFile("delete-me.txt")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		_, err = ws.ReadFile("delete-me.txt")
		if err == nil {
			t.Fatal("expected file does not exist")
		}
	})

	t.Run("manages error for nonexisting file", func(t *testing.T) {
		err := ws.DeleteFile("does-not-exist.txt")

		if err != nil {
			t.Fatal("expected no error, got %w", err)
		}
	})
}

func TestResolvePath(t *testing.T) {
	ws, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	t.Run("accepts valid relative path", func(t *testing.T) {
		path, err := ws.ResolvePath("internal/app.go")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if !strings.HasSuffix(filepath.ToSlash(path), "/internal/app.go") {
			t.Fatalf("unexpected resolved path: %s", path)
		}
	})

	t.Run("rejects absolute path", func(t *testing.T) {
		var absolute string

		if filepath.Separator == '\\' {
			absolute = `C:\outside\file.txt`
		} else {
			absolute = "/outside/file.txt"
		}

		_, err := ws.ResolvePath(absolute)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("rejects path escaping workspace", func(t *testing.T) {
		_, err := ws.ResolvePath("../outside.txt")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("rejects empty path", func(t *testing.T) {
		_, err := ws.ResolvePath("")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
