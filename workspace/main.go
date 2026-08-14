package main

import (
	"fmt"
	"os"
	"strconv"
)

// calculateFactorial calcula el factorial de n de forma recursiva.
// Se utiliza int64 para manejar números factoriales potencialmente grandes.
func calculateFactorial(n int64) int64 {
	// Caso base: el factorial de 0 o 1 es 1.
	if n == 0 || n == 1 {
		return 1
	}
	// Paso recursivo: n * factorial(n-1).
	return n * calculateFactorial(n-1)
}

func main() {
	// Inicializar n con el valor por defecto de 10.
	n := int64(10)

	// Verificar si se proporcionó un argumento desde la línea de comandos.
	if len(os.Args) > 1 {
		// Intentar convertir el argumento a int64.
		inputStr := os.Args[1]
		parsedN, err := strconv.ParseInt(inputStr, 10, 64)

		if err != nil {
			// Si hay un error de conversión, imprimir un mensaje y usar el valor por defecto.
			fmt.Printf("Error: El argumento '%s' no es un número entero válido. Usando n=%d por defecto.\n", inputStr, n)
		} else if parsedN < 0 {
			// Si el número es negativo, imprimir un mensaje y usar el valor por defecto.
			fmt.Printf("Error: El número debe ser no negativo. Usando n=%d por defecto.\n", n)
		} else {
			// Si la conversión es exitosa y el número es válido, asignar a n.
			n = parsedN
		}
	}

	// Calcular el factorial de n.
	result := calculateFactorial(n)

	// Imprimir el resultado.
	fmt.Printf("El factorial de %d es: %d\n", n, result)
}