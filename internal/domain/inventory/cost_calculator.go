package inventory

import "github.com/shopspring/decimal"

// CostCalculator implementa la lógica de costo promedio ponderado (servicio de dominio).
// NuevoCosto = ((StockActual * CostoActual) + (CantEntrada * CostoEntrada)) / (StockActual + CantEntrada)
func CostCalculator(stockActual, costoActual, cantEntrada, costoEntrada decimal.Decimal) decimal.Decimal {
	sum := stockActual.Add(cantEntrada)
	if sum.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}
	num := stockActual.Mul(costoActual).Add(cantEntrada.Mul(costoEntrada))
	return num.Div(sum)
}
