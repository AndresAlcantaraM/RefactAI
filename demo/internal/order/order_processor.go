package orders

import (
	"fmt"
	"time"
)

type Order struct {
	ID     string
	Amount float64
}

// CalcularTotalPedido no supera los 5 parámetros (solo recibe 2),
// pero supera ampliamente las 10 líneas. Debe activar solo "function_too_long".
func CalcularTotalPedido(order Order, impuesto float64) (float64, error) {
	fmt.Printf("Calculando total para la orden: %s\n", order.ID)

	if order.Amount <= 0 {
		return 0, fmt.Errorf("monto inválido")
	}

	subtotal := order.Amount
	impuestoCalculado := subtotal * impuesto

	fmt.Println("Aplicando descuentos de temporada...")
	descuento := 0.0
	if subtotal > 100 {
		descuento = 10.0
	}

	total := subtotal + impuestoCalculado - descuento
	fmt.Printf("Fecha de cálculo: %s\n", time.Now().Format(time.RFC3339))

	return total, nil
}
