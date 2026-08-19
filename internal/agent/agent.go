package agent

import (
	"context"
	"fmt"
	"refactai/internal/action"
	"refactai/internal/executor"
	"refactai/internal/prompt"
)

const maxAttempts = 3

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

	var execution executor.Result

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		act := action.New(code)

		execution, err = a.executor.Run(ctx, act)
		if err == nil {
			return Result{
				Task:      task,
				Plan:      plan,
				Code:      code,
				Execution: execution,
			}, nil
		}

		if ctx.Err() != nil {
			return Result{
				Task:      task,
				Plan:      plan,
				Code:      code,
				Execution: execution,
			}, fmt.Errorf("action execution cancelled: %w", ctx.Err())
		}

		if attempt == maxAttempts {
			return Result{
				Task:      task,
				Plan:      plan,
				Code:      code,
				Execution: execution,
			}, fmt.Errorf("action failed after %d attempts: %w", maxAttempts, err)
		}

		feedbackPrompt := a.prompts.BuildFeedbackAction(task, plan, code, execution.Stdout, execution.Stderr, execution.ExitCode)

		code, err = a.llm.Generate(ctx, feedbackPrompt)
		if err != nil {
			return Result{
				Task:      task,
				Plan:      plan,
				Code:      code,
				Execution: execution,
			}, fmt.Errorf("failed to generate corrective action: %w", err)
		}
	}

	return Result{}, fmt.Errorf("agent reached an unexpected state")
}
