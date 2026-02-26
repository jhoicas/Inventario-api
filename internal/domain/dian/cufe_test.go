package dian_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tu-usuario/inventory-pro/internal/domain/dian"
)

// ──────────────────────────────────────────────────────────────────────────────
// TestCalculateCufe valida que el cálculo SHA-384 del CUFE produce el hash
// exacto esperado para parámetros conocidos.
//
// Este test es el "canario en la mina" de la integración DIAN: si alguien
// modifica inadvertidamente la cadena de concatenación, el algoritmo o el
// formato de los montos, el test falla inmediatamente y el Dockerfile rechaza
// la imagen antes de llegar a producción.
//
// Vector de prueba calculado manualmente con SHA-384:
//
//	Cadena = NumFac + FecFac + ValFac + CodImp01 + ValImp01 + CodImp04 + ValImp04 +
//	         CodImp03 + ValImp03 + ValPag + NitOfe + DocAdq + ClTec + TipoAmb
//	       = "SETP990000000" + "2023-11-29" + "1000000.00" +
//	         "01" + "190000.00" + "04" + "0.00" + "03" + "0.00" +
//	         "1190000.00" + "900123456" + "800987654" +
//	         "fc8eac422eba16e22ffd8c6f94b3f40a6e38162c354673d3a603956897890cd" + "2"
// ──────────────────────────────────────────────────────────────────────────────

const (
	testCufeExpected = "f5693bff411776a0c3536bba5df32491df2ffc101a8ff4810cdfc04368b8a9286dc0d5c578fa2344e119d118947a0c4c"

	testNitOfe  = "900123456"
	testDocAdq  = "800987654"
	testClTec   = "fc8eac422eba16e22ffd8c6f94b3f40a6e38162c354673d3a603956897890cd"
	testFecFac  = "2023-11-29"
	testNumFac  = "SETP990000000"
	testTipoAmb = "2"
)

func TestCalculateCufe_VectorExacto(t *testing.T) {
	svc := dian.NewCufeCalculatorService()

	params := &dian.CufeParams{
		NumFac:    testNumFac,
		FecFac:    testFecFac,
		ValFac:    decimal.NewFromFloat(1_000_000),
		ValImp_01: decimal.NewFromFloat(190_000),
		ValImp_04: decimal.Zero,
		ValImp_03: decimal.Zero,
		ValPag:    decimal.NewFromFloat(1_190_000),
		NitOfe:    testNitOfe,
		DocAdq:    testDocAdq,
		ClTec:     testClTec,
		TipoAmb:   testTipoAmb,
	}

	cufe, err := svc.Calculate(params)
	require.NoError(t, err, "Calculate no debe retornar error con parámetros válidos")
	assert.Equal(t, testCufeExpected, cufe,
		"El CUFE debe coincidir exactamente con el vector SHA-384 de referencia DIAN")
}

// TestCalculateCufe_DeterministaIgual verifica que llamar Calculate dos veces
// con los mismos parámetros produce siempre el mismo hash (idempotente).
func TestCalculateCufe_DeterministaIgual(t *testing.T) {
	svc := dian.NewCufeCalculatorService()
	params := buildTestParams()

	cufe1, err1 := svc.Calculate(params)
	cufe2, err2 := svc.Calculate(params)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, cufe1, cufe2, "El mismo input siempre debe producir el mismo CUFE")
}

// TestCalculateCufe_DiferenteNumFac verifica que cambiar el número de factura
// produce un hash distinto (sensibilidad al input).
func TestCalculateCufe_DiferenteNumFac(t *testing.T) {
	svc := dian.NewCufeCalculatorService()

	p1 := buildTestParams()
	p2 := buildTestParams()
	p2.NumFac = "SETP990000001" // solo cambia el número

	cufe1, _ := svc.Calculate(p1)
	cufe2, _ := svc.Calculate(p2)

	assert.NotEqual(t, cufe1, cufe2,
		"Facturas con números distintos deben tener CUFEs distintos")
}

// TestCalculateCufe_TipoAmbienteAfectaHash verifica que producción (TipoAmb=1)
// y pruebas (TipoAmb=2) producen hashes diferentes.
func TestCalculateCufe_TipoAmbienteAfectaHash(t *testing.T) {
	svc := dian.NewCufeCalculatorService()

	pPruebas := buildTestParams()
	pPruebas.TipoAmb = "2"

	pProduccion := buildTestParams()
	pProduccion.TipoAmb = "1"

	cufePruebas, _ := svc.Calculate(pPruebas)
	cufeProduccion, _ := svc.Calculate(pProduccion)

	assert.NotEqual(t, cufePruebas, cufeProduccion,
		"Los CUFEs de ambiente pruebas y producción deben ser distintos")
}

// ── Errores de validación ─────────────────────────────────────────────────────

func TestCalculateCufe_ErrorSiNilParams(t *testing.T) {
	svc := dian.NewCufeCalculatorService()
	_, err := svc.Calculate(nil)
	assert.Error(t, err, "Calculate con nil debe retornar error")
}

func TestCalculateCufe_ErrorSiNumFacVacio(t *testing.T) {
	svc := dian.NewCufeCalculatorService()
	p := buildTestParams()
	p.NumFac = ""
	_, err := svc.Calculate(p)
	assert.Error(t, err, "Calculate sin NumFac debe retornar error")
}

func TestCalculateCufe_ErrorSiNitOfeVacio(t *testing.T) {
	svc := dian.NewCufeCalculatorService()
	p := buildTestParams()
	p.NitOfe = ""
	_, err := svc.Calculate(p)
	assert.Error(t, err, "Calculate sin NitOfe debe retornar error")
}

func TestCalculateCufe_ErrorSiClTecVacia(t *testing.T) {
	svc := dian.NewCufeCalculatorService()
	p := buildTestParams()
	p.ClTec = ""
	_, err := svc.Calculate(p)
	assert.Error(t, err, "Calculate sin ClTec debe retornar error")
}

// TestCalculateCufe_LongitudHash valida que el hash SHA-384 tenga exactamente
// 96 caracteres hexadecimales (384 bits / 4 bits por nibble = 96 nibbles).
func TestCalculateCufe_LongitudHash(t *testing.T) {
	svc := dian.NewCufeCalculatorService()
	cufe, err := svc.Calculate(buildTestParams())
	require.NoError(t, err)
	assert.Len(t, cufe, 96, "El CUFE debe tener 96 caracteres hexadecimales (SHA-384)")
}

// ── helper ────────────────────────────────────────────────────────────────────

func buildTestParams() *dian.CufeParams {
	return &dian.CufeParams{
		NumFac:    testNumFac,
		FecFac:    testFecFac,
		ValFac:    decimal.NewFromFloat(1_000_000),
		ValImp_01: decimal.NewFromFloat(190_000),
		ValImp_04: decimal.Zero,
		ValImp_03: decimal.Zero,
		ValPag:    decimal.NewFromFloat(1_190_000),
		NitOfe:    testNitOfe,
		DocAdq:    testDocAdq,
		ClTec:     testClTec,
		TipoAmb:   testTipoAmb,
	}
}
