package main

import (
	"fmt"      
	"io"
	"net/http" // se utiliza solo en el tester para poder hacer requests HTTP al servidor
	"time"
	"sort"
	"sync"
)

type Result struct {
	Latency    time.Duration
	StatusCode int
	Error      error
}

func worker(url string, wg *sync.WaitGroup, results chan<- Result) {
	defer wg.Done()

	start := time.Now()
	resp, error := http.Get(url)
	if error != nil {
		results <- Result{Error: error}
		return
	}
	defer resp.Body.Close()

	_, error = io.Copy(io.Discard, resp.Body)
	elapsed := time.Since(start)

	if error != nil {
		results <- Result{Error: error}
		return
	}

	results <- Result{Latency: elapsed, StatusCode: resp.StatusCode}
}

func main() {
	url := "http://localhost:8080/status"
	numRequests := 100
	concurrency := 10

	fmt.Printf("Iniciando prueba de carga contra: %s\n", url)
	fmt.Printf("Total de peticiones: %d\n", numRequests)
	fmt.Printf("Nivel de concurrencia: %d\n", concurrency)
	fmt.Println("---------------------------------")

	var wg sync.WaitGroup

	results := make(chan Result, numRequests)
	
	wg.Add(concurrency)
	
	taskQueue := make(chan string, numRequests)

	for i := 0; i < concurrency; i++ {
		go func() {
			for reqURL := range taskQueue {
				// El worker real hace la petición
				start := time.Now()
				resp, error := http.Get(reqURL)
				var result Result
				if error != nil {
					result = Result{Error: error}
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
				results <- result
			}
			// El worker le dice al WaitGroup que ha terminado.
			wg.Done()
		}()
	}
	// Enviar las peticiones a la cola de tareas
	totalStart := time.Now()

	for i := 0; i < numRequests; i++ {
		taskQueue <- url
	}

	close(taskQueue)
	
	wg.Wait()
	
	close(results)

	// Procesar los resultados
	totalElapsed := time.Since(totalStart)
	processResults(results, totalElapsed, numRequests)

}


func processResults(results <-chan Result, totalElapsed time.Duration, numRequests int) {
	var latencies []time.Duration // Lista de todas las latencias
	var errors int
	var status200 int

	for result := range results {
		if result.Error != nil {
			fmt.Printf("Error en petición: %v\n", result.Error)
			errors++
		} else {
			latencies = append(latencies, result.Latency)
			if result.StatusCode == 200 {
				status200++
			}
		}
	}

	// --- Calcular Estadísticas ---

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	rps := float64(numRequests) / totalElapsed.Seconds()

	// Calcular 
	p50 := latencies[int(float64(len(latencies))*0.50)] // Mediana
	p95 := latencies[int(float64(len(latencies))*0.95)]
	p99 := latencies[int(float64(len(latencies))*0.99)]

	// reporte
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