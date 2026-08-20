package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"refactai/internal/agent"
	"refactai/internal/analyzer"
	"refactai/internal/comparator"
	"refactai/internal/config"
	"refactai/internal/executor"
	"refactai/internal/llm"
	"refactai/internal/prompt"
	"refactai/internal/validator"
	"refactai/internal/workspace"
	"strings"
)

func main() {

	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <project-path> <task>\n", os.Args[0])
		os.Exit(1)
	}

	projectPath := os.Args[1]
	task := os.Args[2]

	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	gemini, err := llm.NewGemini(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}

	promptBuilder := prompt.NewBuilder()

	workspacePath, err := os.MkdirTemp("", "refactai-workspace-*")
	if err != nil {
		log.Fatalf("failed to create temporary workspace: %v", err)
	}

	defer os.RemoveAll(workspacePath)

	if err := workspace.CopyDir(projectPath, workspacePath); err != nil {
		log.Fatalf("failed to copy project to workspace: %v", err)
	}

	ws, err := workspace.New(workspacePath)
	if err != nil {
		log.Fatal(err)
	}

	analyzerClient, err := analyzer.New(ws)
	if err != nil {
		log.Fatal(err)
	}

	exec, err := executor.New(ws)
	if err != nil {
		log.Fatal(err)
	}

	validatorClient, err := validator.New(ws)
	if err != nil {
		log.Fatal(err)
	}

	refactAgent := agent.New(
		gemini,
		promptBuilder,
		analyzerClient,
		exec,
		ws,
		validatorClient,
	)

	comparatorClient := comparator.New()

	log.Printf("TASK:\n%s\n", task)

	result, err := refactAgent.Run(ctx, task)
	if err != nil {
		log.Printf("Agent execution failed: %v", err)
		log.Printf("PLAN:\n%s\n", result.Plan)
		log.Printf("CODE:\n%s\n", result.Code)
		log.Printf("STDOUT:\n%s\n", result.Execution.Stdout)
		log.Printf("STDERR:\n%s\n", result.Execution.Stderr)
		log.Printf("EXIT CODE: %d\n", result.Execution.ExitCode)
		log.Printf("VALIDATION STDOUT:\n%s\n", result.Validation.Stdout)
		log.Printf("VALIDATION STDERR:\n%s\n", result.Validation.Stderr)
		log.Printf("VALIDATION EXIT CODE: %d\n", result.Validation.ExitCode)
		log.Printf("VALIDATION SKIPPED: %t\n", result.Validation.Skipped)
		return
	}

	log.Printf("PLAN:\n%s\n", result.Plan)
	log.Printf("CODE:\n%s\n", result.Code)
	log.Printf("STDOUT:\n%s\n", result.Execution.Stdout)
	log.Printf("STDERR:\n%s\n", result.Execution.Stderr)
	log.Printf("EXIT CODE: %d\n", result.Execution.ExitCode)
	log.Printf("VALIDATION STDOUT:\n%s\n", result.Validation.Stdout)
	log.Printf("VALIDATION STDERR:\n%s\n", result.Validation.Stderr)
	log.Printf("VALIDATION EXIT CODE: %d\n", result.Validation.ExitCode)
	log.Printf("VALIDATION SKIPPED: %t\n", result.Validation.Skipped)

	changes, err := comparatorClient.Compare(projectPath, ws.Root())
	if err != nil {
		log.Fatalf("failed to generate diff: %v", err)
	}

	if len(changes) == 0 {
		log.Println("No changes detected.")
		return
	}

	log.Printf("CHANGES DETECTED (%d file(s)):\n", len(changes))
	for _, change := range changes {
		fmt.Printf("\n--- [%s] File: %s ---\n", change.Type, change.Path)
		fmt.Println(change.Diff)
	}

	fmt.Print("\nApply these changes to the project? [y/N]: ")

	var answer string
	fmt.Scanln(&answer)

	if strings.EqualFold(answer, "y") ||
		strings.EqualFold(answer, "yes") {
		if err := comparatorClient.Apply(
			projectPath,
			ws.Root(),
			changes,
		); err != nil {
			log.Fatalf("failed to apply changes: %v", err)
		}

		log.Println("Changes applied successfully.")
	} else {
		log.Println("Changes discarded.")
	}
}
