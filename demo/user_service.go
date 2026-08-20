package service

import "fmt"

type UserService struct{}

// ProcesarRegistro superará las 10 líneas de longitud y tiene 6 parámetros (> 5).
// Debe activar AMBOS hallazgos: "function_too_long" y "too_many_parameters".
func (s *UserService) ProcesarRegistro(nombre, email, password, direccion, telefono, codigoPostal string) error {
	fmt.Println("Iniciando validación de datos...")
	if nombre == "" {
		return fmt.Errorf("el nombre es obligatorio")
	}
	if email == "" {
		return fmt.Errorf("el email es obligatorio")
	}
	if password == "" {
		return fmt.Errorf("el password es obligatorio")
	}
	fmt.Println("Guardando en la base de datos...")
	fmt.Printf("Usuario registrado: %s (%s)\n", nombre, email)
	return nil
}
