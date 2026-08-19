package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Workspace struct {
	root string
}

func New(root string) (*Workspace, error) {
	if root == "" {
		return nil, fmt.Errorf("workspace root cannot be empty")
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve workspace root: %w", err)
	}

	if err := os.MkdirAll(absoluteRoot, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	return &Workspace{
		root: absoluteRoot,
	}, nil
}

func (w *Workspace) Root() string {
	return w.root
}

func (w *Workspace) WriteFile(path string, content []byte) error {
	fullPath, err := w.ResolvePath(path)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	if err := os.WriteFile(fullPath, content, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (w *Workspace) ReadFile(path string) ([]byte, error) {
	fullPath, err := w.ResolvePath(path)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return content, nil
}

func (w *Workspace) ListFiles() ([]string, error) {
	var files []string

	err := filepath.WalkDir(w.root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(w.root, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		files = append(files, filepath.ToSlash(relativePath))

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list workspace files: %w", err)
	}

	return files, nil
}

func (w *Workspace) DeleteFile(path string) error {
	fullPath, err := w.ResolvePath(path)
	if err != nil {
		return err
	}

	if err := os.Remove(fullPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

func (w *Workspace) ResolvePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}

	cleanPath := filepath.Clean(path)

	if cleanPath == "." || cleanPath == "" {
		return "", fmt.Errorf("invalid workspace path")
	}

	fullPath := filepath.Join(w.root, cleanPath)
	relativePath, err := filepath.Rel(w.root, fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace path: %w", err)
	}

	if relativePath == ".." || len(relativePath) >= 3 && relativePath[:3] == ".."+string(os.PathSeparator) {
		return "", fmt.Errorf("path escapes workspace: %s", path)
	}

	return fullPath, nil
}
