package agent

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"refactai/internal/action"
	"refactai/internal/analyzer"
	"refactai/internal/executor"
	"refactai/internal/prompt"
	"refactai/internal/validator"
	"refactai/internal/workspace"
	"sort"
)

const maxAttempts = 3
const maxContextFileSize = 100 * 1024

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
	fileContents, err := a.readWorkspaceContext(files)
	if err != nil {
		return Result{}, fmt.Errorf("failed to read workspace context: %w", err)
	}
	baseline, err := a.snapshotWorkspace()
	if err != nil {
		return Result{}, fmt.Errorf("failed to snapshot workspace: %w", err)
	}
	planPrompt := a.prompts.BuildPlan(task, findings, files, fileContents)
	plan, err := a.llm.Generate(ctx, planPrompt)
	if err != nil {
		return Result{}, fmt.Errorf("failed to generate plan: %w", err)
	}
	codePrompt := a.prompts.BuildCode(task, plan, findings, files, fileContents)
	code, err := a.llm.Generate(ctx, codePrompt)
	if err != nil {
		return Result{}, fmt.Errorf("failed to generate code: %w", err)
	}
	var execution executor.Result
	var validation validator.Result
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		fmt.Printf("********************** ATTEMPT %d **********************\n", attempt)
		act := action.New(code)
		execution, err = a.executor.Run(ctx, act)
		if err != nil {
			if ctx.Err() != nil {
				return Result{Task: task, Plan: plan, Code: code, Execution: execution, Validation: validation},
					fmt.Errorf("action execution cancelled: %w", ctx.Err())
			}
			if attempt == maxAttempts {
				return Result{Task: task, Plan: plan, Code: code, Execution: execution, Validation: validation},
					fmt.Errorf("action failed after %d attempts: %w", maxAttempts, err)
			}
			code, err = a.generateCorrectiveAction(ctx, task, plan, code, execution, validation)
			if err != nil {
				return Result{Task: task, Plan: plan, Code: code, Execution: execution, Validation: validation}, err
			}
			continue
		}

		validation, err = a.validator.Validate(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return Result{Task: task, Plan: plan, Code: code, Execution: execution, Validation: validation},
					fmt.Errorf("validation cancelled: %w", ctx.Err())
			}
			if attempt == maxAttempts {
				return Result{Task: task, Plan: plan, Code: code, Execution: execution, Validation: validation},
					fmt.Errorf("validation failed after %d attempts: %w", maxAttempts, err)
			}
			code, err = a.generateCorrectiveAction(ctx, task, plan, code, execution, validation)
			if err != nil {
				return Result{Task: task, Plan: plan, Code: code, Execution: execution, Validation: validation}, err
			}
			continue
		}

		current, snapErr := a.snapshotWorkspace()
		if snapErr != nil {
			return Result{Task: task, Plan: plan, Code: code, Execution: execution, Validation: validation},
				fmt.Errorf("failed to snapshot workspace: %w", snapErr)
		}
		if snapshotsEqual(baseline, current) {
			if attempt == maxAttempts {
				return Result{Task: task, Plan: plan, Code: code, Execution: execution, Validation: validation},
					fmt.Errorf("action produced no changes to the workspace after %d attempts", maxAttempts)
			}
			code, err = a.generateNoChangeCorrectiveAction(ctx, task, plan, code, files, fileContents)
			if err != nil {
				return Result{Task: task, Plan: plan, Code: code, Execution: execution, Validation: validation}, err
			}
			continue
		}

		return Result{Task: task, Plan: plan, Code: code, Execution: execution, Validation: validation}, nil
	}
	return Result{}, fmt.Errorf("agent reached an unexpected state")
}

func (a *Agent) readWorkspaceContext(files []string) (map[string]string, error) {
	sortedFiles := append([]string(nil), files...)
	sort.Strings(sortedFiles)

	contents := make(map[string]string)

	for _, file := range sortedFiles {
		if !isContextFile(file) {
			continue
		}

		content, err := a.workspace.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", file, err)
		}

		if len(content) > maxContextFileSize {
			log.Printf(
				"WARNING: %s is %d bytes, exceeds the %d byte context limit; its content will be omitted from the LLM prompt",
				file, len(content), maxContextFileSize,
			)
			contents[file] = fmt.Sprintf(
				"[FILE TOO LARGE TO INCLUDE: %d bytes, exceeds the %d byte limit. Content omitted — do not assume you know this file's contents; avoid modifying it unless strictly necessary.]",
				len(content), maxContextFileSize,
			)
			continue
		}

		contents[file] = string(content)
	}

	return contents, nil
}

func isContextFile(path string) bool {
	ext := filepath.Ext(path)

	switch ext {
	case ".go",
		".mod",
		".sum",
		".md",
		".txt",
		".json",
		".yaml",
		".yml":
		return true
	default:
		return false
	}
}

func (a *Agent) generateCorrectiveAction(ctx context.Context, task string, plan string, previousCode string,
	execution executor.Result, validation validator.Result) (string, error) {
	files, err := a.workspace.ListFiles()
	if err != nil {
		return "", fmt.Errorf("failed to list workspace files for feedback: %w", err)
	}
	fileContents, err := a.readWorkspaceContext(files)
	if err != nil {
		return "", fmt.Errorf("failed to read workspace context for feedback: %w", err)
	}
	feedbackPrompt := a.prompts.BuildFeedbackAction(task, plan, previousCode, execution, validation, files, fileContents)
	code, err := a.llm.Generate(ctx, feedbackPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate corrective action: %w", err)
	}
	return code, nil
}

func (a *Agent) generateNoChangeCorrectiveAction(ctx context.Context, task string, plan string, previousCode string,
	files []string, fileContents map[string]string) (string, error) {
	feedbackPrompt := a.prompts.BuildNoChangeFeedback(task, plan, previousCode, files, fileContents)
	code, err := a.llm.Generate(ctx, feedbackPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate no-change corrective action: %w", err)
	}
	return code, nil
}

func (a *Agent) snapshotWorkspace() (map[string]string, error) {
	files, err := a.workspace.ListFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace files: %w", err)
	}
	snapshot := make(map[string]string, len(files))
	for _, file := range files {
		content, err := a.workspace.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", file, err)
		}
		snapshot[file] = string(content)
	}
	return snapshot, nil
}

func snapshotsEqual(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for path, content := range left {
		other, ok := right[path]
		if !ok || other != content {
			return false
		}
	}
	return true
}
