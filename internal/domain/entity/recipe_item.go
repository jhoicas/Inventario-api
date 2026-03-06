package entity

import "github.com/shopspring/decimal"

// RecipeItem representa una línea de receta (BOM): producto terminado ↔ materia prima.
type RecipeItem struct {
	ProductID        string
	RawMaterialID    string
	QuantityRequired decimal.Decimal
	WastePercentage  decimal.Decimal

	// Para cálculos de costo se suele necesitar el costo de la materia prima:
	RawMaterial *RawMaterial
}

