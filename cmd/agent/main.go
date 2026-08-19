package main

import (
	"context"
	"log"
	"refactai/internal/agent"
	"refactai/internal/config"
	"refactai/internal/executor"
	"refactai/internal/llm"
	"refactai/internal/prompt"
	"refactai/internal/workspace"
)

func main() {
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

	ws, err := workspace.New("./tmp-workspace")
	if err != nil {
		log.Fatal(err)
	}

	exec, err := executor.New(ws)
	if err != nil {
		log.Fatal(err)
	}

	refactAgent := agent.New(
		gemini,
		promptBuilder,
		exec,
	)

	task := "Generate a Go program that prints Hello World!"

	log.Printf("TASK:\n%s\n", task)

	result, err := refactAgent.Run(ctx, task)
	if err != nil {
		log.Printf("Agent execution failed: %v", err)
		log.Printf("PLAN:\n%s\n", result.Plan)
		log.Printf("CODE:\n%s\n", result.Code)
		log.Printf("STDOUT:\n%s\n", result.Execution.Stdout)
		log.Printf("STDERR:\n%s\n", result.Execution.Stderr)
		log.Printf("EXIT CODE: %d\n", result.Execution.ExitCode)
		return
	}

	log.Printf("PLAN:\n%s\n", result.Plan)
	log.Printf("CODE:\n%s\n", result.Code)
	log.Printf("STDOUT:\n%s\n", result.Execution.Stdout)
	log.Printf("STDERR:\n%s\n", result.Execution.Stderr)
	log.Printf("EXIT CODE: %d\n", result.Execution.ExitCode)
}
