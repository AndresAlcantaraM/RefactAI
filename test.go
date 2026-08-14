package main

import (
	"bytes"
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"regexp"

	"github.com/joho/godotenv"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"google.golang.org/genai"

	"strings"
)

type Config struct {
	GeminiApiKey string
	Port         string
}

func LoadConfig() *Config {
	_ = godotenv.Load()
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("Error, no se encontró la api key.")
	}

	port := "8080"

	return &Config{
		GeminiApiKey: apiKey,
		Port:         port,
	}
}

func callLLM(cfg *Config, ctx context.Context, prompt, model string) (string, error) {

	client, err := genai.NewClient(ctx, nil)

	if err != nil {
		log.Fatalf("Hubo un error al intentar crear el cliente de Gemini: %v.", err)
	}

	fmt.Printf("Servidor listo en el puerto %s usando la API Key configurada.\n", cfg.Port)

	resp, err := client.Models.GenerateContent(ctx, model, genai.Text(prompt), nil)

	if err != nil {
		log.Fatalf("Hubo un error al enviar la petición al cliente de Gemini: %v", err)
	}

	fmt.Printf("La respuesta del modelo es: \n%v\n", resp.Text())

	return resp.Text(), nil

}

func isGoValid(code string) (bool, error) {
	fset := token.NewFileSet()

	_, err := parser.ParseFile(fset, "llm_output.go", code, parser.AllErrors)

	if err != nil {
		return false, err
	}

	return true, nil
}

func cleanLLMProgram(code string) string {
	re := regexp.MustCompile("(?s)```(?:go)?\\s*\n?(.*?)\n?```")
	matches := re.FindStringSubmatch(code)

	var program string

	if len(matches) > 1 {
		program = matches[1]
	} else {
		program = code
	}

	return strings.TrimSpace(program)

}

func saveLLMProgram(code, name string) (string, error) {

	workspace := "workspace"

	if err := os.MkdirAll(workspace, 0755); err != nil {
		return "", err
	}

	filePath := filepath.Join(workspace, name+".go")

	if err := os.WriteFile(filePath, []byte(code), 0600); err != nil {
		return "", err
	}

	return filePath, nil
}

func execLLMProgram(code string) (string, error) {

	var outBuffer bytes.Buffer

	i := interp.New((interp.Options{Stdout: &outBuffer}))

	if err := i.Use(stdlib.Symbols); err != nil {
		log.Fatal(err)
	}

	_, err := i.Eval(code)

	if err != nil {
		log.Fatal(err)
	}

	return outBuffer.String(), err
}

func execTestsProgram(workspace string) (string, error) {
	var outBuffer bytes.Buffer

	cmd := exec.Command("go", "test", "-v", ".")
	cmd.Dir = workspace
	cmd.Stdout = &outBuffer
	cmd.Stderr = &outBuffer

	err := cmd.Run()

	return outBuffer.String(), err
}

func main() {

	task := `Genera un programa que calcule el factorial de n con recursividad. Si no se incluye un input para n, n=10 por default.`

	prompt_step_0 := `
	Eres un experto en código Go. Tu tarea es leer la petición del usuario y realizar un plan paso a paso para satisfacerla.

	Restricciones:

	1. Céntrate únicamente en lenguaje Go.
	2. NO GENERES CÓDIGO. Tu tarea es planificar.
	3. Centrate en buenas prácticas de programación en el lenguaje.
	4. La tarea del usuario estará justo después del marcador "TAREA:"
	5. No generes ningún otro comentario además del plan (e.g., Nada de Claro, a continuación... solo responde el plan directamente).
	
	TAREA: 
	`

	prompt_step_1 := `
	Eres un generador de código en lenguaje Go. Tu tarea es generar código en Go según la petición del usuario y el plan generado previamente.

	Restricciones:

	1. Utiliza lenguaje Go siempre, empieza desde el package main
	2. No incluyas tics ni markers indicando el lenguaje (e.g., '''java)
	3. No generes ningún otro comentario además del programa (e.g., Nada de Claro, a continuación... solo responde el código directamente).
	4. Asegúrate de seguir la estructura del plan.
	5. Encontrarás la petición del usuario después del marcador "TAREA: " y el plan después del marcador "PLAN: ".

	TAREA: 
	`

	prompt_step_2 := `
	Eres un experto en pruebas en el lenguaje Go. Tu tarea es generar un programa de pruebas para un contexto, siguiendo buenas prácticas.

	Restricciones:
	
	1. Utiliza lenguaje Go siempre, empieza desde el package main.
	2. No incluyas tics ni markers indicando el lenguaje (e.g., '''java)
	3. No generes ningún otro comentario además del programa (e.g., Nada de Claro, a continuación... solo responde el código directamente).
	4. Asegúrate de testear todas las funcionalidades.
	5. Asegúrate de incluir logs/outputs que se puedan mostrar (e.g., All tests passed!)
	6. Encontrarás la petición del usuario después del marcador "TAREA: ", el plan después del marcador "PLAN: ", el programa desarrollado después del marcador "PROGRAMA: " y el output del programa correspondiente después del marcador "OUTPUT: ".
	7. El programa de tests que generes será ubicado en el mismo paquete y directorio que el programa que incluye el desarrollo de la función.
	8. El programa de tests es solo para el factorial de 10, no para ningún otro número ni lleva input.

	TAREA: 
	`

	cfg := LoadConfig()
	ctx := context.Background()

	plan, _ := callLLM(cfg, ctx, prompt_step_0+task, "gemini-2.5-flash")

	resp, _ := callLLM(cfg, ctx, prompt_step_1+task+"PLAN: "+plan, "gemini-2.5-flash")

	program := cleanLLMProgram(resp)

	status := "Not valid"

	valid, err := isGoValid(program)

	if valid {
		status = "Valid"
	}

	if err != nil {
		log.Fatalf("Hubo un error al intentar parsear el programa de Go: %v.\n", err)
	}

	filePath, err := saveLLMProgram(program, "main")

	if err != nil {
		log.Fatalf("Hubo un error al guardar el programa: %v.\n", err)
	}

	fmt.Printf("El programa fue guardado en %v.\n", filePath)

	fmt.Printf("El bloque de código generado por el LLM es %v.\n", status)

	fmt.Println("Ejecutando código...")

	output, err := execLLMProgram(program)
	fmt.Println("El output del programa del modelo es:")
	fmt.Println(output)

	resp, _ = callLLM(cfg, ctx, prompt_step_2+task+"PLAN: "+plan+"\nPROGRAM: "+program+"\nOUTPUT: "+output, "gemini-2.5-flash")

	test := cleanLLMProgram(resp)

	test_status := "Not valid"

	valid, err = isGoValid(test)

	if valid {
		test_status = "Valid"
	}

	if err != nil {
		log.Fatalf("Hubo un error al validar el programa de tests: %v.", err)
	}

	filePath, err = saveLLMProgram(test, "main_test")

	if err != nil {
		log.Fatalf("Hubo un error al guardar el programa de tests: %v.\n", err)
	}

	fmt.Printf("El programa fue guardado en %v.\n", filePath)

	fmt.Printf("El bloque de pruebas generado por el LLM es %v.\n", test_status)

	fmt.Println("Ejecutando pruebas...")

	output, err = execTestsProgram("workspace")
	fmt.Println("El output del programa de tests es:")
	fmt.Println(output)

}
