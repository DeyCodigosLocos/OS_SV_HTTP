// para correr este codigo use la siguiente instruccion como ejemplo:
// go run tester/main.go -url http://localhost:8080/status -n 1000 -c 50
// donde url es el endpoint a probar, n es el numero total de peticiones y 
// c es el nivel de concurrencia, osease, el numero de usuarios virtuales simultaneos
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http" // se utiliza solo en el tester para poder hacer requests HTTP al servidor
	"sort"
	"sync"
	"time"
)

type Result struct {
	Latency    time.Duration
	StatusCode int
	Error      error
}

// worker es el "Usuario" Virtual (Worker)
func worker(id int, taskQueue <-chan string, resultsChan chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()

	for reqURL := range taskQueue {
		start := time.Now()
		resp, err := http.Get(reqURL)
		var result Result

		if err != nil {
			result = Result{Error: err}
		} else {
			_, errBody := io.Copy(io.Discard, resp.Body)
			elapsed := time.Since(start)
			resp.Body.Close()

			if errBody != nil {
				result = Result{Error: errBody}
			} else {
				result = Result{Latency: elapsed, StatusCode: resp.StatusCode}
			}
		}

		resultsChan <- result
	}
}

func main() {
	// Configuración de la Prueba (CON FLAGS)
	urlPtr := flag.String("url", "http://localhost:8080/status", "URL del endpoint a probar")
	nPtr := flag.Int("n", 100, "Número total de peticiones")
	cPtr := flag.Int("c", 10, "Nivel de concurrencia (usuarios simultáneos)")

	flag.Parse()

	url := *urlPtr
	numRequests := *nPtr
	concurrency := *cPtr

	fmt.Printf("Iniciando prueba de carga contra: %s\n", url)
	fmt.Printf("Total de peticiones: %d\n", numRequests)
	fmt.Printf("Nivel de concurrencia: %d\n", concurrency)
	fmt.Println("---------------------------------")

	var wg sync.WaitGroup
	resultsChan := make(chan Result, numRequests)
	taskQueue := make(chan string, numRequests)

	// Lanzar los workers
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker(i, taskQueue, resultsChan, &wg)
	}

	totalStart := time.Now()

	// Enviar "trabajos"
	for i := 0; i < numRequests; i++ {
		taskQueue <- url
	}
	close(taskQueue)

	// Esperar
	wg.Wait()
	totalElapsed := time.Since(totalStart)
	close(resultsChan)

	// Procesar
	processResults(resultsChan, totalElapsed, numRequests)
}

// processResults es el "Estadístico"
func processResults(resultsChan <-chan Result, totalElapsed time.Duration, numRequests int) {
	var latencies []time.Duration
	var errors int
	var status200 int

	for result := range resultsChan {
		if result.Error != nil {
			errors++
		} else {
			latencies = append(latencies, result.Latency)
			if result.StatusCode == 200 {
				status200++
			}
		}
	}

	// Protección contra crash
	if len(latencies) == 0 {
		fmt.Println("\n--- Resultados de la Prueba ---")
		fmt.Printf("Tiempo total: %s\n", totalElapsed.Round(time.Millisecond))
		fmt.Printf("Peticiones fallidas (errores): %d\n", errors)
		fmt.Println("No se recibieron respuestas exitosas para calcular latencias.")
		return
	}

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	rps := float64(numRequests) / totalElapsed.Seconds()

	// Calcular Percentiles
	p50 := latencies[int(float64(len(latencies))*0.50)]
	p95 := latencies[int(float64(len(latencies))*0.95)]
	p99 := latencies[int(float64(len(latencies))*0.99)]

	// Imprimir Reporte
	fmt.Println("\n--- Resultados de la Prueba ---")
	fmt.Printf("Tiempo total: %s\n", totalElapsed.Round(time.Millisecond))
	fmt.Printf("Throughput (RPS): %.2f\n", rps)
	fmt.Printf("Peticiones exitosas (200): %d\n", status200)
	fmt.Printf("Errores: %d\n", errors)
	fmt.Println("\n--- Percentiles de Latencia ---")
	fmt.Printf("p50 (Mediana): %s\n", p50.Round(time.Microsecond))
	fmt.Printf("p95: %s\n", p95.Round(time.Microsecond))
	fmt.Printf("p99: %s\n", p99.Round(time.Microsecond))
	fmt.Println("---------------------------------")
}