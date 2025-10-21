package tasks

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"math/big"
	"time"
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
	start := time.Now()

	// Establece la precisión en bits (aprox. 3.32 bits por dígito)
	prec := uint(digits * 4)
	

	// Constantes del algoritmo de Chudnovsky
	a := big.NewFloat(13591409).SetPrec(prec)
	b := big.NewFloat(545140134).SetPrec(prec)
	
	sum := new(big.Float).SetPrec(prec).SetFloat64(0)
	
	sixKFact := big.NewFloat(1).SetPrec(prec)
	threeKFact := big.NewFloat(1).SetPrec(prec)
	kFact := big.NewFloat(1).SetPrec(prec)

	neg1pow := 1.0
	for k := 0; k < digits/14+1; k++ { // 14 dígitos por iteración aprox.
		// (-1)^k * (6k)! * (13591409 + 545140134k)
		t1 := new(big.Float).SetPrec(prec).Copy(sixKFact)
		t2 := new(big.Float).SetPrec(prec).Mul(b, new(big.Float).SetPrec(prec).SetFloat64(float64(k)))
		t2.Add(t2, a)
		t1.Mul(t1, t2)
		if neg1pow < 0 {
			t1.Neg(t1)
		}

		// (3k)! * (k!)^3 * (640320^(3k/2))
		t3 := new(big.Float).SetPrec(prec).Mul(threeKFact, new(big.Float).SetPrec(prec).Mul(kFact, kFact))
		t3.Mul(t3, kFact)
		t4 := new(big.Float).SetPrec(prec).SetFloat64(math.Pow(640320, float64(3*k)))
		t4.Mul(t4, big.NewFloat(1)) // Placeholder para mantener precisión
		t3.Mul(t3, t4)

		term := new(big.Float).SetPrec(prec).Quo(t1, t3)
		sum.Add(sum, term)

		// Update factorials para siguiente iteración
		sixKFact.Mul(sixKFact, new(big.Float).SetFloat64(float64((6*k+1)*(6*k+2)*(6*k+3)*(6*k+4)*(6*k+5)*(6*k+6))))
		threeKFact.Mul(threeKFact, new(big.Float).SetFloat64(float64((3*k+1)*(3*k+2)*(3*k+3))))
		kFact.Mul(kFact, new(big.Float).SetFloat64(float64(k+1)))
		neg1pow *= -1
	}

	// Calcular pi
	cubeRoot := new(big.Float).SetPrec(prec).SetFloat64(math.Pow(640320, 1.5))
	pi := new(big.Float).SetPrec(prec).Mul(sum, big.NewFloat(12))
	pi.Quo(cubeRoot, pi)

	elapsed := time.Since(start)
	return fmt.Sprintf("%.*f", digits, pi) + fmt.Sprintf(" (%.2fs)", elapsed.Seconds())
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
