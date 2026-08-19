package agent

import (
	"context"
	"fmt"
	"refactai/internal/action"
	"refactai/internal/analyzer"
	"refactai/internal/executor"
	"refactai/internal/prompt"
	"refactai/internal/validator"
	"refactai/internal/workspace"
)

const maxAttempts = 3

type LLM interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

type Agent struct {
	llm       LLM
	prompts   *prompt.Builder
	analyzer  *analyzer.Analyzer
	executor  *executor.Executor
	workspace *workspace.Workspace
	validator *validator.Validator
}

type Result struct {
	Task       string
	Plan       string
	Code       string
	Execution  executor.Result
	Validation validator.Result
}

func New(llmClient LLM, promptBuilder *prompt.Builder, analyzerClient *analyzer.Analyzer, executorClient *executor.Executor,
	workspace *workspace.Workspace, validator *validator.Validator) *Agent {
	return &Agent{
		llm:       llmClient,
		prompts:   promptBuilder,
		analyzer:  analyzerClient,
		executor:  executorClient,
		workspace: workspace,
		validator: validator,
	}
}

func (a *Agent) Run(ctx context.Context, task string) (Result, error) {
	if task == "" {
		return Result{}, fmt.Errorf("task cannot be empty")
	}

	files, err := a.workspace.ListFiles()
	if err != nil {
		return Result{}, fmt.Errorf("failed to list workspace files: %v", err)
	}

	findings, err := a.analyzer.Analyze()
	if err != nil {
		return Result{}, fmt.Errorf("failed to analyze workspace: %w", err)
	}

	planPrompt := a.prompts.BuildPlan(task, findings, files)

	plan, err := a.llm.Generate(ctx, planPrompt)
	if err != nil {
		return Result{}, fmt.Errorf("failed to generate plan: %w", err)
	}

	codePrompt := a.prompts.BuildCode(task, plan, findings, files)

	code, err := a.llm.Generate(ctx, codePrompt)
	if err != nil {
		return Result{}, fmt.Errorf("failed to generate code: %w", err)
	}

	var execution executor.Result
	var validation validator.Result

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		act := action.New(code)

		execution, err = a.executor.Run(ctx, act)
		if err != nil {
			if ctx.Err() != nil {
				return Result{
					Task:       task,
					Plan:       plan,
					Code:       code,
					Execution:  execution,
					Validation: validation,
				}, fmt.Errorf("action execution cancelled: %w", ctx.Err())
			}

			if attempt == maxAttempts {
				return Result{
					Task:       task,
					Plan:       plan,
					Code:       code,
					Execution:  execution,
					Validation: validation,
				}, fmt.Errorf("action failed after %d attempts: %w", maxAttempts, err)
			}

			code, err = a.generateCorrectiveAction(ctx, task, plan, code, execution, validation)
			if err != nil {
				return Result{
					Task:       task,
					Plan:       plan,
					Code:       code,
					Execution:  execution,
					Validation: validation,
				}, err
			}

			continue
		}

		validation, err := a.validator.Validate(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return Result{
					Task:       task,
					Plan:       plan,
					Code:       code,
					Execution:  execution,
					Validation: validation,
				}, fmt.Errorf("validation cancelled: %w", ctx.Err())
			}

			if attempt == maxAttempts {
				return Result{
					Task:       task,
					Plan:       plan,
					Code:       code,
					Execution:  execution,
					Validation: validation,
				}, fmt.Errorf("validation failed after %d attempts: %w", maxAttempts, err)
			}

			code, err = a.generateCorrectiveAction(ctx, task, plan, code, execution, validation)
			if err != nil {
				return Result{
					Task:       task,
					Plan:       plan,
					Code:       code,
					Execution:  execution,
					Validation: validation,
				}, err
			}

			continue
		}

		return Result{
			Task:       task,
			Plan:       plan,
			Code:       code,
			Execution:  execution,
			Validation: validation,
		}, nil
	}
	return Result{}, fmt.Errorf("agent reached an unexpected state")
}

func (a *Agent) generateCorrectiveAction(ctx context.Context, task string, plan string, previousCode string,
	execution executor.Result, validation validator.Result) (string, error) {
	feedbackPrompt := a.prompts.BuildFeedbackAction(task, plan, previousCode, execution, validation)

	code, err := a.llm.Generate(ctx, feedbackPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate corrective action: %w", err)
	}

	return code, nil
}
