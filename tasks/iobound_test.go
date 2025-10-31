package tasks

import (
	"bufio"   
	"os"
	"path/filepath"
	"sort"    
	"strconv" 
	"strings"
	"testing"
)
// Helper function to create a temporary test file
func createTempFile(t *testing.T, content string) string {
	t.Helper() // Marca esto como una función helper

	// Crear un directorio temporal para esta prueba
	dir := t.TempDir()
	path := filepath.Join(dir, "testfile.txt")
	
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("No se pudo crear el archivo temporal: %v", err)
	}
	return path
}

func TestWordCount(t *testing.T) {
	content := "hola mundo\nqué tal\nadios\n"
	path := createTempFile(t, content)

	lines, words, bytes, err := WordCount(path)
	if err != nil {
		t.Fatalf("WordCount devolvió un error inesperado: %v", err)
	}

	if lines != 3 {
		t.Errorf("WordCount.lines = %d; se esperaba 3", lines)
	}
	if words != 5 {
		t.Errorf("WordCount.words = %d; se esperaba 5", words)
	}
	if bytes != 26 {
		t.Errorf("WordCount.bytes = %d; se esperaba 26", bytes)
	}
}

// TestGrep prueba la función Grep
func TestGrep(t *testing.T) {
	content := "linea uno\nlinea dos con patron\nlinea tres\nlinea cuatro con patron\n"
	path := createTempFile(t, content)

	count, lines, err := Grep(path, "patron")
	if err != nil {
		t.Fatalf("Grep devolvió un error inesperado: %v", err)
	}

	if count != 2 {
		t.Errorf("Grep.count = %d; se esperaba 2", count)
	}
	if len(lines) != 2 {
		t.Errorf("Grep.lines (count) = %d; se esperaba 2", len(lines))
	}
	if lines[0] != "linea dos con patron" {
		t.Errorf("Grep.lines[0] incorrecto: %s", lines[0])
	}
}

// TestHashFile prueba la función HashFile
func TestHashFile(t *testing.T) {
	content := "hola mundo"
	path := createTempFile(t, content)
	
	expectedHash := "0b894166d3336435c800bea36ff21b29eaa801a52f584c006c49289a0dcf6e2f"
	
	hash, err := HashFile(path)
	if err != nil {
		t.Fatalf("HashFile devolvió un error inesperado: %v", err)
	}

	if hash != expectedHash {
		t.Errorf("HashFile = %s; se esperaba %s", hash, expectedHash)
	}
}

// TestCompress (gzip) prueba la función Compress
func TestCompress(t *testing.T) {
	content := "un texto para comprimir"
	path := createTempFile(t, content)

	outName, size, err := Compress(path, "gzip")
	if err != nil {
		t.Fatalf("Compress(gzip) devolvió un error: %v", err)
	}

	if !strings.HasSuffix(outName, ".gz") {
		t.Errorf("El nombre de salida no termina en .gz: %s", outName)
	}
	if size <= 0 {
		t.Errorf("El tamaño del archivo comprimido es <= 0: %d", size)
	}
}



// TestSortFile prueba los algoritmos "quick" y "merge" de SortFile
func TestSortFile(t *testing.T) {
	// Contenido de prueba: números desordenados
	content := "5\n2\n10\n1\n8\n"
	
	// --- Prueba 1: Algoritmo "quick" ---
	pathQuick := createTempFile(t, content)
	sortedPathQuick, _, err := SortFile(pathQuick, "quick")
	if err != nil {
		t.Fatalf("SortFile(quick) devolvió un error: %v", err)
	}
	checkFileIsSorted(t, sortedPathQuick)

	// --- Prueba 2: Algoritmo "merge" ---
	// (Usamos el mismo archivo pequeño; probará la ruta de "1 solo chunk")
	pathMerge := createTempFile(t, content)
	sortedPathMerge, _, err := SortFile(pathMerge, "merge")
	if err != nil {
		t.Fatalf("SortFile(merge) devolvió un error: %v", err)
	}
	checkFileIsSorted(t, sortedPathMerge)
}

// checkFileIsSorted es una función helper para leer un archivo de números
// y verificar si están ordenados.
func checkFileIsSorted(t *testing.T, path string) {
	t.Helper()
	
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("checkFileIsSorted: No se pudo abrir el archivo %s: %v", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var nums []int
	for scanner.Scan() {
		n, err := strconv.Atoi(scanner.Text())
		if err != nil {
			if scanner.Text() == "" {
				continue
			}
			t.Fatalf("checkFileIsSorted: El archivo contiene un no-número: %v", err)
		}
		nums = append(nums, n)
	}

	if !sort.IntsAreSorted(nums) {
		t.Errorf("checkFileIsSorted: El archivo %s no está ordenado. Contenido: %v", path, nums)
	}
	
	// 5, 2, 10, 1, 8 -> 1, 2, 5, 8, 10
	if len(nums) != 5 || nums[0] != 1 || nums[4] != 10 {
		t.Errorf("checkFileIsSorted: Contenido inesperado. Se esperaban 5 elementos ordenados de 1 a 10. Se obtuvo: %v", nums)
	}
}