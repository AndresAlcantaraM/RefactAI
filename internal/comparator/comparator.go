package comparator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
)

type ChangeType string

const (
	Added    ChangeType = "added"
	Modified ChangeType = "modified"
	Deleted  ChangeType = "deleted"
)

type Change struct {
	Path string
	Type ChangeType
	Diff string
}

type Comparator struct{}

func New() *Comparator {
	return &Comparator{}
}

func (c *Comparator) Compare(originalRoot, workspaceRoot string) ([]Change, error) {
	originalFiles, err := listFiles(originalRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to list original files: %w", err)
	}

	workspaceFiles, err := listFiles(workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace files: %w", err)
	}

	allFiles := make(map[string]struct{})

	for path := range originalFiles {
		allFiles[path] = struct{}{}
	}

	for path := range workspaceFiles {
		allFiles[path] = struct{}{}
	}

	var changes []Change

	for path := range allFiles {
		originalPath := filepath.Join(
			originalRoot,
			filepath.FromSlash(path),
		)

		workspacePath := filepath.Join(
			workspaceRoot,
			filepath.FromSlash(path),
		)

		_, originalExists := originalFiles[path]
		_, workspaceExists := workspaceFiles[path]

		switch {
		case !originalExists && workspaceExists:
			content, err := os.ReadFile(workspacePath)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to read added file %s: %w",
					path,
					err,
				)
			}

			changes = append(changes, Change{
				Path: path,
				Type: Added,
				Diff: unifiedDiff(path, "", string(content)),
			})

		case originalExists && !workspaceExists:
			content, err := os.ReadFile(originalPath)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to read deleted file %s: %w",
					path,
					err,
				)
			}

			changes = append(changes, Change{
				Path: path,
				Type: Deleted,
				Diff: unifiedDiff(path, string(content), ""),
			})

		case originalExists && workspaceExists:
			originalContent, err := os.ReadFile(originalPath)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to read original file %s: %w",
					path,
					err,
				)
			}

			workspaceContent, err := os.ReadFile(workspacePath)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to read workspace file %s: %w",
					path,
					err,
				)
			}

			if string(originalContent) == string(workspaceContent) {
				continue
			}

			changes = append(changes, Change{
				Path: path,
				Type: Modified,
				Diff: unifiedDiff(
					path,
					string(originalContent),
					string(workspaceContent),
				),
			})
		}
	}

	return changes, nil
}

func listFiles(root string) (map[string]struct{}, error) {
	files := make(map[string]struct{})

	err := filepath.WalkDir(
		root,
		func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if entry.IsDir() {
				return nil
			}

			relativePath, err := filepath.Rel(root, path)
			if err != nil {
				return fmt.Errorf(
					"failed to get relative path: %w",
					err,
				)
			}

			files[filepath.ToSlash(relativePath)] = struct{}{}

			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	return files, nil
}

func unifiedDiff(path, before, after string) string {
	edits := myers.ComputeEdits(span.URIFromPath(path), before, after)
	unified := gotextdiff.ToUnified("a/"+path, "b/"+path, before, edits)
	return fmt.Sprint(unified)
}

func (c *Comparator) Apply(originalRoot string, workspaceRoot string, changes []Change) error {
	for _, change := range changes {
		originalPath := filepath.Join(
			originalRoot,
			filepath.FromSlash(change.Path),
		)

		workspacePath := filepath.Join(
			workspaceRoot,
			filepath.FromSlash(change.Path),
		)

		switch change.Type {
		case Added, Modified:
			content, err := os.ReadFile(workspacePath)
			if err != nil {
				return fmt.Errorf(
					"failed to read changed file %s: %w",
					change.Path,
					err,
				)
			}

			if err := os.MkdirAll(filepath.Dir(originalPath), 0755); err != nil {
				return fmt.Errorf(
					"failed to create parent directory for %s: %w",
					change.Path,
					err,
				)
			}

			if err := os.WriteFile(originalPath, content, 0644); err != nil {
				return fmt.Errorf(
					"failed to apply change to %s: %w",
					change.Path,
					err,
				)
			}

		case Deleted:
			if err := os.Remove(originalPath); err != nil {
				return fmt.Errorf(
					"failed to delete %s: %w",
					change.Path,
					err,
				)
			}

		default:
			return fmt.Errorf(
				"unsupported change type %q for %s",
				change.Type,
				change.Path,
			)
		}
	}

	return nil
}
