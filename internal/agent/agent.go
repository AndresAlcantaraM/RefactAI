package agent

import (
	"context"
	"fmt"
	"refactai/internal/action"
	"refactai/internal/executor"
	"refactai/internal/prompt"
)

type LLM interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

type Agent struct {
	llm      LLM
	prompts  *prompt.Builder
	executor *executor.Executor
}

type Result struct {
	Task      string
	Plan      string
	Code      string
	Execution executor.Result
}

func New(llmClient LLM, promptBuilder *prompt.Builder, executorClient *executor.Executor) *Agent {
	return &Agent{
		llm:      llmClient,
		prompts:  promptBuilder,
		executor: executorClient,
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

	act := action.New(code)

	execution, err := a.executor.Run(ctx, act)
	if err != nil {
		return Result{
			Task:      task,
			Plan:      plan,
			Code:      code,
			Execution: execution,
		}, fmt.Errorf("failed to execute action: %w", err)
	}

	return Result{
		Task:      task,
		Plan:      plan,
		Code:      code,
		Execution: execution,
	}, nil
}
