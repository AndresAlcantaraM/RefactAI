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

func (b *Builder) BuildPlan(task string, findings []analyzer.Finding, files []string,
	fileContents map[string]string) string {
	return fmt.Sprintf(`
		You are an expert software engineer.

		Analyze the user's task and the existing workspace structure.

		Create a concrete implementation plan.

		Rules:
		- Focus on the existing workspace.
		- Do not generate code.
		- Consider the analyzer findings.
		- Do not propose unnecessary changes.
		- Follow best practices.

		USER TASK:
		%s

		WORKSPACE FILES:
		%s

		ANALYZER FINDINGS:
		%s

		WORKSPACE FILE CONTENTS:
		%s

		`, task, formatFiles(files), formatFindings(findings), formatFileContents(files, fileContents))
}

func (b *Builder) BuildCode(task string, plan string, findings []analyzer.Finding, files []string,
	fileContents map[string]string) string {
	return fmt.Sprintf(`
	You are an expert Go software engineer working as a CodeAct agent.
	IMPORTANT:
	You are generating an ACTION PROGRAM, not the final refactored repository code.
	Your generated Go source will be saved as action.go and compiled together
	with a second file, refactai_tools.go, which is ALREADY PRESENT in the
	same package and provides these helper functions you can call directly
	(do not redeclare them, do not redefine them, just call them):
		func replaceExact(path, oldStr, newStr string) error
			Replaces oldStr with newStr in the file at path. Returns an error
			if oldStr is not found, or found more than once in the file.
		func createFile(path, content string) error
			Creates a brand new file. Returns an error if the file already exists.
		func deleteFile(path string) error
			Deletes an existing file. Returns an error if it does not exist.
		func replaceFunction(path, functionName, newSource string) error
			Finds the top-level function or method named functionName in path
			using Go's AST (not text matching) and replaces its entire
			declaration with newSource. newSource may contain more than one
			function declaration — e.g. the modified function plus new helper
			functions you introduce — all of them will be inserted at the
			original function's position. The result is automatically
			gofmt'd and syntax-checked. Returns an error if functionName is
			not found or the resulting file is not valid Go.
	Your action.go must define package main and func main(), and should call
	these helper functions to perform the requested changes. You can use
	loops, conditionals, and call these functions multiple times to compose
	several changes in a single action. Check every returned error and, on
	any failure, print it and exit with a non-zero status (e.g. log.Fatal),
	so failures are visible instead of silent.
	The current working directory when action.go runs is the root of the repository.

	- Whenever the change modifies an existing function's body, logic,
	  or signature, PREFER replaceFunction over replaceExact: you only
	  need to write the NEW function (plus any new helper functions),
	  not reproduce the old one character-by-character.
	- Use replaceExact only for small, non-function-level edits (a
	  single import line, a single call site, a comment) where copying
	  a short, unique snippet is reliable.

	ACTION PROGRAM RULES:
	- Use Go only.
	- Start with package main.
	- Prefer calling replaceExact / createFile / deleteFile over writing
	  raw os.WriteFile calls with reconstructed file contents.
	- The generated program must actually modify the existing repository.
	- Do NOT output the final contents of main.go, utils.go, or other repository files as your response.
	- Do NOT replace the repository with a new implementation.
	- Do NOT generate a standalone example or demonstration program.
	- Do NOT only print the changes that should be made.
	- Use relative paths from the workspace root.
	- Do not use absolute paths.
	- Do not modify files outside the workspace.
	- Do not delete files unless explicitly required by the implementation plan.
	- Do not use Markdown code fences.
	CRITICAL ANTI-PATTERN TO AVOID:
	Do NOT define a Go string literal that contains a nearly complete
	reconstruction of an existing file (for example, a multi-line backtick
	string that repeats most of the original file's declarations, imports,
	or function bodies) and then write it verbatim with os.WriteFile.
	This is considered a full-file rewrite and is NOT acceptable, even if
	the resulting file would be syntactically correct.
	If you catch yourself writing a string literal that reproduces more than
	a few lines of an existing file, STOP and use replaceExact instead,
	copying old_str exactly from WORKSPACE FILE CONTENTS below.
	REFACTORING SCOPE:
	The analyzer findings define the primary scope of this refactoring.
	- Prefer addressing the reported analyzer findings.
	- Prefer modifying only files and functions related to those findings.
	- Do not introduce unrelated architectural changes.
	- Do not remove unrelated functionality.
	- Do not change public APIs unless explicitly required.
	- Preserve existing behavior unless the user task explicitly requires a behavior change.
	- Make the smallest reasonable change necessary to satisfy the task.
	EXAMPLE:
	package main
    import "log"
    func main() {
        newSource := `+"`"+`func Greet() string {
    return "Greet"
}`+"`"+`
        if err := replaceFunction("main.go", "Hello", newSource); err != nil {
            log.Fatal(err)
        }
    }
	USER TASK:
	%s
	IMPLEMENTATION PLAN:
	%s
	CODE ANALYSIS FINDINGS:
	%s
	WORKSPACE FILES:
	%s
	WORKSPACE FILE CONTENTS:
	%s
	`, task, plan, formatFindings(findings), formatFiles(files), formatFileContents(files, fileContents))
}

func (b *Builder) BuildFeedbackAction(task string, plan string, previousCode string, execution executor.Result,
	validation validator.Result, files []string, fileContents map[string]string) string {
	return fmt.Sprintf(`
	You are an expert Go software engineer working as a CodeAct agent.
	The previously generated action.go did not produce an acceptable result.
	Remember: action.go is compiled together with refactai_tools.go, which
	already provides these functions you can call:
		func replaceExact(path, oldStr, newStr string) error
		func createFile(path, content string) error
		func deleteFile(path string) error
		func replaceFunction(path, functionName, newSource string) error
	Analyze the execution and validation feedback below, and generate a
	corrected action.go. The CURRENT WORKSPACE FILE CONTENTS reflect the
	real state of the repository right now (some of your previous changes
	may have already been applied) — base any replaceExact old_str on that
	current content, not on your previous assumptions.
	Rules:
	- Use Go only.
	- Start with package main.
	- Preserve the original task and implementation plan.
	- Fix the problem identified by the execution or validation feedback.
	- Prefer calling replaceExact / createFile / deleteFile over raw file rewrites.
	- Generate only the complete source code for action.go.
	- Do not use Markdown code fences.
	- Do not include explanations outside the source code.
	- Use relative paths from the workspace root.
	- Do not modify files outside the workspace.
	- Check every returned error and exit non-zero on failure.
	- Make the smallest reasonable changes necessary to satisfy the task.
	USER TASK:
	%s
	IMPLEMENTATION PLAN:
	%s
	PREVIOUS ACTION.GO:
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
	CURRENT WORKSPACE FILES:
	%s
	CURRENT WORKSPACE FILE CONTENTS:
	%s
	`,
		task, plan, previousCode,
		execution.Stdout, execution.Stderr, execution.ExitCode,
		validation.Skipped, validation.Stdout, validation.Stderr, validation.ExitCode,
		formatFiles(files), formatFileContents(files, fileContents),
	)
}

func (b *Builder) BuildNoChangeFeedback(task string, plan string, previousCode string, files []string,
	fileContents map[string]string) string {
	return fmt.Sprintf(`
	You are an expert Go software engineer working as a CodeAct agent.
	Your previous action.go compiled and ran successfully, and all
	validations passed, but it did NOT modify any file in the repository.
	Remember action.go is compiled together with refactai_tools.go, which
	provides replaceExact(path, oldStr, newStr), createFile(path, content), 
	deleteFile(path) and replaceFunction(path, functionName, newSource string) — all returning a Go error.
	This usually means one of the following happened:
	- You called replaceExact but ignored its returned error, so a failed
	  match was silently skipped instead of stopping the program.
	- old_str passed to replaceExact did not match the actual file content
	  exactly (whitespace, indentation, or text differs from WORKSPACE FILE
	  CONTENTS below).
	- The program never actually called any tool function.
	Generate a corrected action.go that actually applies a visible change,
	checks every returned error, and exits non-zero on failure so problems
	are never silent again.
	Rules:
	- Use Go only.
	- Start with package main.
	- Base any old_str strictly on the exact WORKSPACE FILE CONTENTS below.
	- Prefer replaceExact for targeted modifications.
	- Do not use Markdown code fences.
	- Do not include explanations outside the source code.
	- Use relative paths from the workspace root.
	- Do not modify files outside the workspace.
	USER TASK:
	%s
	IMPLEMENTATION PLAN:
	%s
	PREVIOUS ACTION.GO (produced no changes):
	%s
	WORKSPACE FILES:
	%s
	WORKSPACE FILE CONTENTS:
	%s
	`,
		task, plan, previousCode, formatFiles(files), formatFileContents(files, fileContents),
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

func formatFileContents(files []string, contents map[string]string) string {
	var builder strings.Builder

	for _, file := range files {
		content, ok := contents[file]
		if !ok {
			continue
		}

		fmt.Fprintf(
			&builder,
			"\n--- %s ---\n%s\n",
			file,
			content,
		)
	}

	return builder.String()
}
