package notification

import "fmt"

// EnviarNotificacionCompleta cabe en menos de 10 líneas,
// pero declara 6 parámetros (> 5). Debe activar solo "too_many_parameters".
func EnviarNotificacionCompleta(destinatario, asunto, cuerpo, prioridad, canal string, intentarReintento bool) bool {
	fmt.Printf("[%s] Enviando a %s vía %s: %s\n", prioridad, destinatario, canal, asunto)
	return true
}

// NotificacionAnonima prueba la firma con tipos agrupados o sin nombres.
func NotificacionAnonima(a, b, c, d, e, f string) {
	_ = a + b + c + d + e + f
}
