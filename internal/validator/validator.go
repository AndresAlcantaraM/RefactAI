package validator

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"refactai/internal/workspace"
)

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Skipped  bool
}

type Validator struct {
	workspace *workspace.Workspace
}

func New(ws *workspace.Workspace) (*Validator, error) {
	if ws == nil {
		return nil, fmt.Errorf("workspace cannot be nil")
	}

	return &Validator{
		workspace: ws,
	}, nil
}

func (v *Validator) Validate(ctx context.Context) (Result, error) {
	hasGoModule, err := v.hasGoModule()
	if err != nil {
		return Result{}, err
	}

	if !hasGoModule {
		return Result{
			Skipped:  true,
			ExitCode: 0,
		}, nil
	}

	var stdout, stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, "go", "test", "./...")
	cmd.Dir = v.workspace.Root()
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	result := Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}

	if err == nil {
		return result, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
	} else {
		result.ExitCode = -1
	}

	return result, fmt.Errorf("validation failed: %w", err)
}

func (v *Validator) hasGoModule() (bool, error) {
	_, err := os.Stat(filepath.Join(v.workspace.Root(), "go.mod"))

	if err == nil {
		return true, nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, fmt.Errorf("failed to check go.mod: %w", err)
}
