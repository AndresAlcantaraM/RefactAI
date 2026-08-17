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
