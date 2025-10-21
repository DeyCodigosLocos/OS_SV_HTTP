package tasks

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/rand"
)

// ---------------------------
// CPU-bound: primalidad/factorización
// ---------------------------

// IsPrime: simple prueba por división hasta sqrt(n).
// Para n grandes y exigencias de performance, podés agregar Miller-Rabin.
func IsPrime(n int64) (bool, error) {
	if n < 2 {
		return false, nil
	}
	if n%2 == 0 {
		return n == 2, nil
	}
	limit := int64(math.Sqrt(float64(n)))
	for d := int64(3); d <= limit; d += 2 {
		if n%d == 0 {
			return false, nil
		}
	}
	return true, nil
}

// Factor: factorización por división trial (retorna slice de pares [prime,count]).
func Factor(n int64) ([][]int64, error) {
	if n < 2 {
		return nil, errors.New("n debe ser >= 2")
	}
	var res [][]int64
	// factor 2
	cnt := int64(0)
	for n%2 == 0 {
		cnt++
		n /= 2
	}
	if cnt > 0 {
		res = append(res, []int64{2, cnt})
	}
	// odd factors
	var f int64 = 3
	limit := func(x int64) int64 { return int64(math.Sqrt(float64(x))) + 1 }
	for n > 1 && f <= limit(n) {
		cnt = 0
		for n%f == 0 {
			cnt++
			n /= f
		}
		if cnt > 0 {
			res = append(res, []int64{f, cnt})
		}
		f += 2
	}
	if n > 1 {
		res = append(res, []int64{n, 1})
	}
	return res, nil
}

// ---------------------------
// PiDigits calcula los primeros n dígitos de π usando el algoritmo Spigot.
// Es iterativo y usa big.Int para precisión arbitraria.
// ---------------------------
func PiDigits(digits int) string {
	if digits <= 0 {
		return ""
	}

	const extra = 10 // pequeños dígitos adicionales para redondeo
	n := (digits * 10 / 3) + extra

	// Inicializar arrays
	arr := make([]int, n)
	for i := range arr {
		arr[i] = 2
	}

	result := make([]byte, 0, digits)
	var carry int

	for i := 0; i < digits; i++ {
		sum := 0
		for j := n - 1; j > 0; j-- {
			sum = arr[j]*10 + carry
			div := (2*j - 1)
			arr[j] = sum % div
			carry = sum / div * j
		}
		arr[0] = carry % 10
		carry /= 10
		result = append(result, byte(carry)+'0')
	}

	// Insertar el punto decimal después del primer dígito
	output := string(result)
	if len(output) > 1 {
		output = output[:1] + "." + output[1:]
	}
	return output
}

// helper: factorial using big.Int
func factorialBig(n int) *big.Int {
	res := big.NewInt(1)
	for i := 2; i <= n; i++ {
		res.Mul(res, big.NewInt(int64(i)))
	}
	return res
}

// ---------------------------
// Mandelbrot: returns matrix[height][width] with iter count until escape (0..maxIter)
// ---------------------------
func Mandelbrot(width, height, maxIter int) ([][]int, error) {
	if width <= 0 || height <= 0 || maxIter <= 0 {
		return nil, errors.New("parametros invalidos")
	}
	// typical view: x from -2.5..1, y from -1..1
	xmin, xmax := -2.5, 1.0
	ymin, ymax := -1.0, 1.0
	result := make([][]int, height)
	for j := 0; j < height; j++ {
		result[j] = make([]int, width)
		y := ymin + (float64(j)/float64(height))*(ymax-ymin)
		for i := 0; i < width; i++ {
			x := xmin + (float64(i)/float64(width))*(xmax-xmin)
			cx, cy := x, y
			var zx, zy float64 = 0, 0
			iter := 0
			for zx*zx+zy*zy <= 4 && iter < maxIter {
				// z = z^2 + c
				nzx := zx*zx - zy*zy + cx
				nzy := 2*zx*zy + cy
				zx, zy = nzx, nzy
				iter++
			}
			result[j][i] = iter
		}
	}
	return result, nil
}

// ---------------------------
// Matrix multiplication -> returns sha256 hex of result matrix flattened
// ---------------------------
func MatrixMul(size int, seed int64) (string, error) {
	if size <= 0 {
		return "", errors.New("size debe ser > 0")
	}
	r := rand.New(rand.NewSource(seed))
	// generate matrices A and B
	A := make([][]int64, size)
	B := make([][]int64, size)
	for i := 0; i < size; i++ {
		A[i] = make([]int64, size)
		B[i] = make([]int64, size)
		for j := 0; j < size; j++ {
			A[i][j] = r.Int63n(1000)
			B[i][j] = r.Int63n(1000)
		}
	}
	// multiply C = A * B (naive)
	C := make([][]int64, size)
	for i := 0; i < size; i++ {
		C[i] = make([]int64, size)
		for j := 0; j < size; j++ {
			var sum int64 = 0
			for k := 0; k < size; k++ {
				sum += A[i][k] * B[k][j]
			}
			C[i][j] = sum
		}
	}
	// flatten and hash
	h := sha256.New()
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			// write int64 as bytes (decimal)
			fmt.Fprintf(h, "%d,", C[i][j])
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
