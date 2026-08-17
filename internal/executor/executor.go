package executor

import (
	"bytes"
	"context"
	"os/exec"
)

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type Executor struct {
	Workspace string
}

func New(workspace string) *Executor {
	return &Executor{
		Workspace: workspace,
	}
}

func (e *Executor) Run(ctx context.Context, command string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = e.Workspace

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}

	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}

	return result, err
}
