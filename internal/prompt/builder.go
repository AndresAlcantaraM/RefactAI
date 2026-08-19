package prompt

import "fmt"

type Builder struct {
	Instructions string
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) BuildPlan(task string) string {
	return fmt.Sprintf(`
	You are an expert Go software engineer.

	Your task is to analyze the user's request and create a step-by-step implementation plan.

	Rules:
	- Focus only on Go.
	- Do not generate code.
	- Follow Go best practices.
	- The plan should be concrete and actionable.
	- Do not include unnecessary explanations.

	USER TASK:
	%s
	`, task)
}

func (b *Builder) BuildCode(task, plan string) string {
	return fmt.Sprintf(`
	You are an expert Go software engineer.

	Generate a Go program that satisfies the user's task.

	Rules:
	- Use Go only.
	- Start with package main.
	- Follow the provided implementation plan.
	- Generate only the source code.
	- Do not use Markdown code fences.
	- Do not include explanations outside the source code.

	USER TASK:
	%s

	IMPLEMENTATION PLAN:
	%s
	`, task, plan)
}

func (b* Builder) BuildFeedbackAction(task, plan, previousCode, stdout, stderr string, exitCode int) string {
	return fmt.Sprintf(`
	You are an expert Go software engineer.

	The previously generated Go program failed during execution.
	Analyze the execution feedback and generate a corrected version.

	Rules:
	- Use Go only.
	- Start with package main.
	- Preserve the original task and implementation plan.
	- Fix the problem identified by the execution feedback.
	- Generate only the complete source code.
	- Do not use Markdown code fences.
	- Do not include explanations outside the source code.

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
	`, task, plan, previousCode, stdout, stderr, exitCode)
}
