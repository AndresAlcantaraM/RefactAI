package prompt

import (
	"fmt"
	"refactai/internal/analyzer"
	"refactai/internal/executor"
	"refactai/internal/validator"
	"strings"
)

type Builder struct {
	Instructions string
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) BuildPlan(task string, findings []analyzer.Finding, files []string) string {
	return fmt.Sprintf(`
		You are an expert software engineer.

		Analyze the user's task and the existing repository structure.

		Create a concrete implementation plan.

		Rules:
		- Focus on the existing repository.
		- Do not generate code.
		- Consider the analyzer findings.
		- Do not propose unnecessary changes.
		- Follow best practices.

		USER TASK:
		%s

		REPOSITORY FILES:
		%s

		ANALYZER FINDINGS:
		%s
		`, task, formatFiles(files), formatFindings(findings))
}

func (b *Builder) BuildCode(task, plan string, findings []analyzer.Finding, files []string) string {
	return fmt.Sprintf(`
		You are an expert Go software engineer working as a CodeAct agent.

		Your task is to generate an executable Go action that performs the requested changes directly on the repository.

		The generated program will be executed from the repository workspace using:

		go run action.go

		The current working directory is the root of the repository.

		Rules:
		- Use Go only.
		- Start with package main.
		- The program must perform the requested repository changes.
		- The program may read and write files using the Go standard library.
		- Use relative paths from the workspace root.
		- Do not use absolute paths.
		- Do not modify files outside the workspace.
		- Do not generate a standalone example or demonstration program.
		- Do not only print the changes that should be made.
		- Actually modify the repository.
		- Do not use Markdown code fences.
		- Generate only executable Go source code.
		- Handle file operation errors explicitly.
		- Do not delete files unless the implementation plan explicitly requires it.
		- Make the smallest reasonable changes necessary to satisfy the task.

		USER TASK:
		%s

		IMPLEMENTATION PLAN:
		%s

		CODE ANALYSIS FINDINGS:
		%s

		WORKSPACE FILES:
		%s
		`, task, plan, formatFindings(findings), formatFiles(files))
}

func (b *Builder) BuildFeedbackAction(task string, plan string, previousCode string, execution executor.Result,
	validation validator.Result) string {

	return fmt.Sprintf(`
	You are an expert Go software engineer.

	The previously generated Go program did not produce an acceptable repository state.
	Analyze the execution and validation feedback and generate a corrected version.

	Rules:
	- Use Go only.
	- Start with package main.
	- Preserve the original task and implementation plan.
	- Fix the problem identified by the execution or validation feedback.
	- Generate only the complete source code.
	- Do not use Markdown code fences.
	- Do not include explanations outside the source code.
	- The program must modify the repository directly.
	- Use relative paths from the workspace root.
	- Do not modify files outside the workspace.
	- Handle file operation errors explicitly.
	- Make the smallest reasonable changes necessary to satisfy the task.

	USER TASK:
	%s

	IMPLEMENTATION PLAN:
	%s

	PREVIOUS CODE:
	%s

	EXECUTION STDOUT:
	%s

	EXECUTION STDERR:
	%s

	EXECUTION EXIT CODE:
	%d

	VALIDATION SKIPPED:
	%t

	VALIDATION STDOUT:
	%s

	VALIDATION STDERR:
	%s

	VALIDATION EXIT CODE:
	%d
	`,
		task,
		plan,
		previousCode,
		execution.Stdout,
		execution.Stderr,
		execution.ExitCode,
		validation.Skipped,
		validation.Stdout,
		validation.Stderr,
		validation.ExitCode,
	)
}

func formatFindings(findings []analyzer.Finding) string {
	if len(findings) == 0 {
		return "No issues were detected by the static analyzer."
	}

	var builder strings.Builder

	for _, finding := range findings {
		fmt.Fprintf(&builder, "- %s:%d [%s] %s\n", finding.File, finding.Line, finding.Type, finding.Message)
	}

	return builder.String()
}

func formatFiles(files []string) string {
	if len(files) == 0 {
		return "No files were found in the workspace."
	}

	var builder strings.Builder

	for _, file := range files {
		fmt.Fprintf(&builder, "- %s\n", file)
	}

	return builder.String()
}
