package main

import (
	"context"
	"log"
	"refactai/internal/agent"
	"refactai/internal/config"
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

	refactAgent := agent.New(
		gemini,
		promptBuilder,
	)

	task := "Generate a Go program that calculates the factorial of a number recursively."

	result, err := refactAgent.Run(ctx, task)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("PLAN:\n %v\n", result.Plan)
	log.Printf("CODE:\n %s\n", result.Code)

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

	workspace, err := workspace.New("./tmp-workspace")
	if err != nil {
		log.Fatal(err)
	}

	err = workspace.WriteFile("main.go", []byte("package main\n\nfunc main() {}\n"))
	if err != nil {
		log.Fatal(err)
	}

	content, err := workspace.ReadFile("main.go")
	if err != nil {
		log.Fatal(err)
	}

	log.Println(string(content))

	files, err := workspace.ListFiles()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("LOS ARCHIVOS SON:\n%s", files)
}
