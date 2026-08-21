package service

import "fmt"

type UserService struct{}

// ProcesarRegistro superará las 10 líneas de longitud y tiene 6 parámetros (> 5).
// Debe activar AMBOS hallazgos: "function_too_long" y "too_many_parameters".
type UserRegistration struct {
	Nombre, Email, Password, Direccion, Telefono, CodigoPostal string
}

func (s *UserService) validateRegistration(u *UserRegistration) error {
	if u.Nombre == "" {
		return fmt.Errorf("el nombre es obligatorio")
	}
	if u.Email == "" {
		return fmt.Errorf("el email es obligatorio")
	}
	if u.Password == "" {
		return fmt.Errorf("el password es obligatorio")
	}
	return nil
}

func (s *UserService) ProcesarRegistro(u UserRegistration) error {
	fmt.Println("Iniciando validación de datos...")
	if err := s.validateRegistration(&u); err != nil {
		return err
	}
	fmt.Println("Guardando en la base de datos...")
	fmt.Printf("Usuario registrado: %s (%s)\n", u.Nombre, u.Email)
	return nil
}
