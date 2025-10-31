package tasks

import (
	"fmt" 
	"testing"
)

// TestIsPrime prueba la función IsPrime con casos básicos
func TestIsPrime(t *testing.T) {
	// Casos de prueba: {número, resultado_esperado}
	testCases := []struct {
		n      int64
		expect bool
	}{
		{0, false},
		{1, false},
		{2, true},
		{4, false},
		{7, true},
		{997, true}, 
		{1000, false},
	}

	for _, tc := range testCases {
		result, err := IsPrime(tc.n)
		if err != nil {
			t.Errorf("IsPrime(%d) devolvió un error: %v", tc.n, err)
		}
		if result != tc.expect {
			t.Errorf("IsPrime(%d) = %v; se esperaba %v", tc.n, result, tc.expect)
		}
	}
}

// TestFactor prueba la función Factor
func TestFactor(t *testing.T) {
	n := int64(360)
	expect := "[[2 3] [3 2] [5 1]]" // Formato simple para comparar

	result, err := Factor(n)
	if err != nil {
		t.Fatalf("Factor(%d) devolvió un error: %v", n, err)
	}

	// Convertir resultado a string para comparación simple
	resultStr := fmt.Sprintf("%v", result)
	if resultStr != expect {
		t.Errorf("Factor(%d) = %s; se esperaba %s", n, resultStr, expect)
	}
}

// TestPiDigits prueba que PiDigits devuelva el número correcto de dígitos
func TestPiDigits(t *testing.T) {
	// Probar 7 dígitos de "3.141592"
	digits := 7
	pi, err := PiDigits(digits)
	if err != nil {
		t.Fatalf("PiDigits(%d) devolvió un error: %v", digits, err)
	}

	// El resultado incluye el tiempo, ej: "3.1415926 (0.00s)"
	// Solo verificamos el prefijo.
	expectedPrefix := "3.141592"
	if len(pi) < len(expectedPrefix) || pi[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("PiDigits(%d) = %s; no comienza con %s", digits, pi, expectedPrefix)
	}
}

// TestMatrixMul prueba que MatrixMul devuelva un hash con la semilla correcta
func TestMatrixMul(t *testing.T) {
	// Usamos un tamaño pequeño para que la prueba sea rápida
	size := 10
	seed := int64(42)

	// Hash esperado para MatrixMul(10, 42)
	expectedHash := "20534e79df92f01e05343e7499cd6809e55a99fea155f7e62351914bf812c803"

	hash, err := MatrixMul(size, seed)
	if err != nil {
		t.Fatalf("MatrixMul(%d, %d) devolvió un error: %v", size, seed, err)
	}

	if hash != expectedHash {
		t.Errorf("MatrixMul(%d, %d) = %s; se esperaba %s", size, seed, hash, expectedHash)
	}
}

// TestMandelbrot prueba la ejecución de Mandelbrot
func TestMandelbrot(t *testing.T) {
	// Usamos un tamaño muy pequeño
	width, height, maxIter := 10, 10, 10

	matrix, err := Mandelbrot(width, height, maxIter)
	if err != nil {
		t.Fatalf("Mandelbrot devolvió un error: %v", err)
	}

	if len(matrix) != height {
		t.Errorf("Mandelbrot altura = %d; se esperaba %d", len(matrix), height)
	}
	if len(matrix[0]) != width {
		t.Errorf("Mandelbrot anchura = %d; se esperaba %d", len(matrix[0]), width)
	}
}