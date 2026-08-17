package main

import (
	"context"
	"fmt"
	"log"
	"refactai/internal/config"
	"refactai/internal/llm"
	"refactai/internal/prompt"
)

func main() {
	ctx := context.Background()
	cfg, err := config.Load()

	builder := prompt.NewBuilder()
	if err != nil {
		log.Fatal(err)
	}

	gemini, err := llm.NewGemini(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}

	task := "Generate a Go program that calculates the factorial of a number recursively."

	planPrompt := builder.BuildPlan(task)

	plan, err := gemini.Generate(ctx, planPrompt)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("PLAN:\n%s\n", plan)

	codePrompt := builder.BuildCode(task, plan)

	code, err := gemini.Generate(ctx, codePrompt)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("CODE:\n%s\n", code)

	/*
		exec := executor.New(".")

		result, err := exec.Run(ctx, "go", "test", "../../workspace")

		if err != nil {
			log.Printf("Command failed: %v", err)
		}

		log.Printf("stdout:\n%s", result.Stdout)
		log.Printf("stderr:\n%s", result.Stderr)
		log.Printf("exit code:\n%d", result.ExitCode)
	*/
}
