package dian

import (
	"fmt"
	"unicode"
)

// pesos para el cálculo del dígito de verificación NIT (Orden Administrativa 4 de 1989, DIAN).
// Se aplican a los 9 primeros dígitos del NIT, de izquierda a derecha.
var nitWeights = [9]int{41, 37, 29, 23, 19, 17, 13, 7, 3}

// ValidateNITVerificationDigit valida que el NIT (con o sin puntos/guiones) tenga
// un dígito de verificación correcto según el algoritmo módulo 11 de la DIAN.
// taxID puede ser "123456789-1", "123.456.789-1" o "1234567891".
func ValidateNITVerificationDigit(taxID string) error {
	digits := extractDigits(taxID)
	if len(digits) < 9 {
		return fmt.Errorf("dian: NIT debe tener al menos 9 dígitos, se encontraron %d", len(digits))
	}
	base := digits[:9]
	var sum int
	for i, d := range base {
		sum += int(d-'0') * nitWeights[i]
	}
	remainder := sum % 11
	var expected byte
	if remainder == 0 || remainder == 1 {
		expected = byte('0' + remainder)
	} else {
		expected = byte('0' + (11 - remainder))
	}
	if len(digits) == 10 {
		if digits[9] != expected {
			return fmt.Errorf("dian: dígito de verificación del NIT inválido: esperado %c, recibido %c", expected, digits[9])
		}
		return nil
	}
	return fmt.Errorf("dian: NIT de persona jurídica debe incluir dígito de verificación (10 dígitos), se recibieron %d", len(digits))
}

// ComputeNITVerificationDigit calcula el dígito de verificación para los 9 primeros dígitos del NIT.
// Útil para completar el NIT antes de enviar a la DIAN.
func ComputeNITVerificationDigit(taxID string) (byte, error) {
	digits := extractDigits(taxID)
	if len(digits) < 9 {
		return 0, fmt.Errorf("dian: se requieren al menos 9 dígitos para calcular el dígito de verificación, se encontraron %d", len(digits))
	}
	base := digits[:9]
	var sum int
	for i, d := range base {
		sum += int(d-'0') * nitWeights[i]
	}
	remainder := sum % 11
	if remainder == 0 || remainder == 1 {
		return byte('0' + remainder), nil
	}
	return byte('0' + (11 - remainder)), nil
}

func extractDigits(s string) []byte {
	var out []byte
	for _, r := range s {
		if unicode.IsDigit(r) {
			out = append(out, byte(r))
		}
	}
	return out
}
