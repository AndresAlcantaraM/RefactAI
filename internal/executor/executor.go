package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"refactai/internal/action"
	"refactai/internal/workspace"
	"runtime"
)

const (
	actionFileName   = "action.go"
	actionBinaryName = "action"
)

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type Executor struct {
	workspace *workspace.Workspace
}

func New(workspace *workspace.Workspace) (*Executor, error) {
	if workspace == nil {
		return nil, fmt.Errorf("workspace canno be nil")
	}

	return &Executor{
		workspace: workspace,
	}, nil
}

func (e *Executor) Run(ctx context.Context, act *action.Action) (Result, error) {

	if act == nil {
		return Result{}, fmt.Errorf("action cannot be nil")
	}

	if act.Code == "" {
		return Result{}, fmt.Errorf("action code cannot be empty")
	}

	if err := e.workspace.WriteFile(actionFileName, []byte(act.Code)); err != nil {
		return Result{}, fmt.Errorf("failed to write action: %w", err)
	}

	binaryName := actionBinaryName
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	binaryPath := filepath.Join(e.workspace.Root(), binaryName)

	defer func() {
		_ = e.workspace.DeleteFile(binaryName)
	}()

	if err := e.build(ctx, binaryName); err != nil {
		return Result{
			Stderr:   err.Error(),
			ExitCode: -1,
		}, fmt.Errorf("failed to build action. %w", err)
	}

	return e.execute(ctx, binaryPath)
}

func (e *Executor) build(ctx context.Context, binaryName string) error {
	cmd := exec.CommandContext(
		ctx,
		"go",
		"build",
		"-o",
		binaryName,
		actionFileName,
	)

	cmd.Dir = e.workspace.Root()

	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) > 0 {
			return fmt.Errorf("%s: %w", output, err)
		}

		return err
	}

	return nil
}

func (e *Executor) execute(ctx context.Context, binaryPath string) (Result, error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Dir = e.workspace.Root()
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

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

	return result, fmt.Errorf("failed to execute action: %w", err)
}
