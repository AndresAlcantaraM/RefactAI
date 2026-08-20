package analyzer

import (
	"testing"

	"refactai/internal/workspace"
)

func TestAnalyze(t *testing.T) {
	ws, err := workspace.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	err = ws.WriteFile("main.go", []byte(`
		package main

		func main() {
			println("hello")
		}

		func tooLong() {
			println("1")
			println("2")
			println("3")
			println("4")
			println("5")
			println("6")
			println("7")
			println("8")
			println("9")
			println("10")
			println("11")
			println("12")
			println("13")
			println("14")
			println("15")
			println("16")
			println("17")
			println("18")
			println("19")
			println("20")
			println("21")
			println("22")
			println("23")
			println("24")
			println("25")
			println("26")
			println("27")
			println("28")
			println("29")
			println("30")
			println("31")
		}`))
	if err != nil {
		t.Fatalf("failed to write Go file: %v", err)
	}

	analyzer, err := New(ws)
	if err != nil {
		t.Fatalf("failed to create analyzer: %v", err)
	}

	findings, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	finding := findings[0]

	if finding.Type != "function_too_long" {
		t.Fatalf("expected function_too_long, got %q", finding.Type)
	}

	if finding.File != "main.go" {
		t.Fatalf("expected main.go, got %q", finding.File)
	}

	if finding.Line == 0 {
		t.Fatal("expected finding line")
	}
}

func TestAnalyzeTooManyParameters(t *testing.T) {
	ws, err := workspace.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	err = ws.WriteFile("main.go", []byte(`
		package main

		func createUser(
			name string,
			email string,
			age int,
			city string,
			country string,
			active bool,
		) {
			println(name, email, age, city, country, active)
		}
		`))
	if err != nil {
		t.Fatalf("failed to write Go file: %v", err)
	}

	analyzer, err := New(ws)
	if err != nil {
		t.Fatalf("failed to create analyzer: %v", err)
	}

	findings, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	if findings[0].Type != "too_many_parameters" {
		t.Fatalf(
			"expected too_many_parameters, got %q",
			findings[0].Type,
		)
	}
}

func TestAnalyzeIgnoresNonGoFiles(t *testing.T) {
	ws, err := workspace.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	if err := ws.WriteFile("README.md", []byte("# RefactAI")); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}

	analyzer, err := New(ws)
	if err != nil {
		t.Fatalf("failed to create analyzer: %v", err)
	}

	findings, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %d", len(findings))
	}
}
