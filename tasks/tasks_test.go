package tasks

import (
	"os"
	"path/filepath"
	"strings" // Importado para TestStatus
	"testing"
	"time"
)

// TestFibonacci prueba la función Fibonacci
func TestFibonacci(t *testing.T) {
	testCases := []struct {
		n      int
		expect int
	}{
		{0, 0},
		{1, 1},
		{7, 13},
		{10, 55},
	}
	for _, tc := range testCases {
		res := Fibonacci(tc.n)
		if res != tc.expect {
			t.Errorf("Fibonacci(%d) = %d; se esperaba %d", tc.n, res, tc.expect)
		}
	}
}

// TestReverse prueba la función Reverse
func TestReverse(t *testing.T) {
	res := Reverse("hola")
	if res != "aloh" {
		t.Errorf("Reverse(\"hola\") = %s; se esperaba \"aloh\"", res)
	}
	res = Reverse("reconocer")
	if res != "reconocer" {
		t.Errorf("Reverse(\"reconocer\") = %s; se esperaba \"reconocer\"", res)
	}
}

// TestToUpper prueba la función ToUpper
func TestToUpper(t *testing.T) {
	res := ToUpper("Hola Mundo")
	if res != "HOLA MUNDO" {
		t.Errorf("ToUpper(\"Hola Mundo\") = %s; se esperaba \"HOLA MUNDO\"", res)
	}
}

// TestHash prueba la función Hash (SHA-256 de un string)
func TestHash(t *testing.T) {
	res := Hash("hola mundo")
	
	// Usamos el hash que tu función genera consistentemente
	expect := "0b894166d3336435c800bea36ff21b29eaa801a52f584c006c49289a0dcf6e2f"
	if res != expect {
		t.Errorf("Hash(\"hola mundo\") = %s; se esperaba %s", res, expect)
	}
}

// TestRandomNumbers prueba la función RandomNumbers
func TestRandomNumbers(t *testing.T) {
	nums := RandomNumbers(10, 0, 100)
	if len(nums) != 10 {
		t.Fatalf("RandomNumbers(10) devolvió %d números; se esperaba 10", len(nums))
	}
	for _, n := range nums {
		if n < 0 || n >= 100 {
			t.Errorf("RandomNumbers(10, 0, 100) devolvió %d, fuera de rango [0, 100)", n)
		}
	}
}

// TestCreateAndDeleteFile prueba el ciclo de vida de CreateFile y DeleteFile
func TestCreateAndDeleteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "testcreate.txt")
	content := "test"
	
	// Prueba CreateFile
	err := CreateFile(path, content, 2)
	if err != nil {
		t.Fatalf("CreateFile devolvió un error: %v", err)
	}
	
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("No se pudo leer el archivo creado: %v", err)
	}

	// Esperamos el contenido con saltos de línea que tu función crea
	expectedContent := "test\ntest\n" 
	if string(data) != expectedContent {
		t.Errorf("Contenido del archivo = %q; se esperaba %q", string(data), expectedContent)
	}

	// Prueba DeleteFile
	err = DeleteFile(path)
	if err != nil {
		t.Fatalf("DeleteFile devolvió un error: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("DeleteFile no borró el archivo: %s", path)
	}
}

// TestStatus prueba la función Status
func TestStatus(t *testing.T) {
	startTime := time.Now().Add(-10 * time.Second) // Simula 10s de uptime
	pid := 12345
	
	statusStr := Status(startTime, pid)
	// Asumimos que el pid (segundo arg) es 'active_tasks' basado en el handler
	expectPrefix := `{"uptime": "10s", "active_tasks": 12345` 

	if !strings.HasPrefix(statusStr, expectPrefix) {
		t.Errorf("Status() = %s; no comienza con %s", statusStr, expectPrefix)
	}
}