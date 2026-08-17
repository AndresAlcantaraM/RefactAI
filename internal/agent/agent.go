package agent

import (
	"context"
	"fmt"
	"refactai/internal/prompt"
)

type LLM interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

type Agent struct {
	llm     LLM
	prompts *prompt.Builder
}

type Result struct {
	Task string
	Plan string
	Code string
}

func New(llmClient LLM, promptBuilder *prompt.Builder) *Agent {
	return &Agent{
		llm:     llmClient,
		prompts: promptBuilder,
	}
}

func (a *Agent) Run(ctx context.Context, task string) (Result, error) {
	if task == "" {
		return Result{}, fmt.Errorf("task cannot be empty")
	}

	planPrompt := a.prompts.BuildPlan(task)

	plan, err := a.llm.Generate(ctx, planPrompt)
	if err != nil {
		return Result{}, fmt.Errorf("failed to generate plan: %w", err)
	}

	codePrompt := a.prompts.BuildCode(task, plan)

	code, err := a.llm.Generate(ctx, codePrompt)
	if err != nil {
		return Result{}, fmt.Errorf("failed to generate code: %w", err)
	}

	return Result{
		Task: task,
		Plan: plan,
		Code: code,
	}, nil
}
