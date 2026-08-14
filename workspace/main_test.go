package main

import (
	"fmt"
	"os"
	"testing"
)

// TestMain es una función especial para configurar y ejecutar pruebas,
// y permite un control sobre el ciclo de vida de las pruebas, incluyendo la impresión de mensajes finales.
func TestMain(m *testing.M) {
	// Ejecuta todas las pruebas en el paquete.
	exitCode := m.Run()

	// Si todas las pruebas pasaron, imprime un mensaje.
	if exitCode == 0 {
		fmt.Println("All tests passed!")
	} else {
		fmt.Println("Some tests failed.")
	}

	// Termina el programa de prueba con el código de salida apropiado.
	os.Exit(exitCode)
}

// TestCalculateFactorial prueba la función calculateFactorial para varios casos.
func TestCalculateFactorial(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected int64
	}{
		{
			name:     "Factorial of 0",
			input:    0,
			expected: 1,
		},
		{
			name:     "Factorial of 1",
			input:    1,
			expected: 1,
		},
		{
			name:     "Factorial of 10",
			input:    10,
			expected: 3628800,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := calculateFactorial(tt.input)
			if actual != tt.expected {
				t.Errorf("Para n=%d, se esperaba %d pero se obtuvo %d", tt.input, tt.expected, actual)
			}
		})
	}
}