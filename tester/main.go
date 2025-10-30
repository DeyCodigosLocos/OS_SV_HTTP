// para correr este codigo use la siguiente instruccion como ejemplo:
// go run tester/main.go -url="http://localhost:8080/jobs/submit?task=pi&digits=1000" -n 50 -c 10
// donde url es el endpoint a probar, n es el numero total de peticiones y
// c es el nivel de concurrencia, osease, el numero de usuarios virtuales simultaneos
package main

import (
	"flag"
	"fmt"
	"net/http" // se utiliza solo en el tester para poder hacer requests HTTP al servidor
	"net/url"
	"sort"
	"sync"
	"time"
	"encoding/json"
)

type Result struct {
	Latency    time.Duration
	StatusCode int
	Error      error
}

// Structs para parsear las respuestas JSON del Job Manager
type SubmitResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
	Error  string `json:"error"`
}

type StatusResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error"`
}

// worker es el "Usuario" Virtual (Worker)
func worker(id int, baseURL string, taskQueue <-chan string, resultsChan chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()

	// El worker vive hasta que la taskQueue se cierra
	for submitURL := range taskQueue {
		totalStart := time.Now() // Inicia cronómetro

		// --- PASO 1: SUBMIT (Enviar Job) ---
		resp, err := http.Get(submitURL)
		if err != nil {
			resultsChan <- Result{Error: err}
			continue
		}

		var submitResp SubmitResponse
		err = json.NewDecoder(resp.Body).Decode(&submitResp)
		resp.Body.Close()
		if err != nil {
			resultsChan <- Result{Error: fmt.Errorf("error decoding submit response: %v", err)}
			continue
		}

		// Manejar error 503 (Backpressure) o 400 (Tarea no encontrada)
		if resp.StatusCode != 200 || submitResp.Error != "" {
			resultsChan <- Result{Error: fmt.Errorf("submit failed (%d): %s", resp.StatusCode, submitResp.Error), StatusCode: resp.StatusCode}
			continue
		}

		jobID := submitResp.JobID
		status := submitResp.Status
		statusURL := fmt.Sprintf("%s/jobs/status?id=%s", baseURL, jobID)

		// --- PASO 2: POLL (Sondear Estado) ---
		const maxPolls = 100 // Límite de seguridad (100 * 200ms = 20s)
		for i := 0; i < maxPolls; i++ {
			if status == "done" || status == "error" || status == "canceled" {
				break // Job terminado, salir del bucle de sondeo
			}

			time.Sleep(200 * time.Millisecond) // Espera entre sondeos

			resp, err = http.Get(statusURL)
			if err != nil {
				resultsChan <- Result{Error: fmt.Errorf("poll failed: %v", err)}
				continue // Sale de este worker, va al siguiente job
			}

			var statusResp StatusResponse
			err = json.NewDecoder(resp.Body).Decode(&statusResp)
			resp.Body.Close()
			if err != nil {
				resultsChan <- Result{Error: fmt.Errorf("error decoding status response: %v", err)}
				continue // Sale de este worker
			}
			status = statusResp.Status
		}

		// --- PASO 3: REPORT (Reportar Resultado) ---
		totalElapsed := time.Since(totalStart) // Detiene cronómetro

		if status == "done" {
			// Éxito
			resultsChan <- Result{Latency: totalElapsed, StatusCode: 200}
		} else if status == "error" || status == "canceled" {
			// Fallo controlado por el servidor
			resultsChan <- Result{Error: fmt.Errorf("job %s finalizó con estado: %s", jobID, status), StatusCode: 500}
		} else {
			// Timeout de nuestro agente (superó maxPolls)
			resultsChan <- Result{Error: fmt.Errorf("job %s superó el timeout del agente (20s)", jobID), StatusCode: 504}
		}
	}
}

func main() {
	// Configuración de la Prueba (CON FLAGS)
	urlPtr := flag.String("url", "http://localhost:8080/jobs/submit?task=pi&digits=100", "URL completa del /jobs/submit (incluyendo task y params)")
	nPtr := flag.Int("n", 100, "Número total de peticiones")
	cPtr := flag.Int("c", 10, "Nivel de concurrencia (usuarios simultáneos)")

	flag.Parse()

	urlSubmit := *urlPtr
	numRequests := *nPtr
	concurrency := *cPtr

	// Extraer el baseURL (ej. "http://localhost:8080") desde la URL de submit
	// para poder construir la URL de /jobs/status
	parsedURL, err := url.Parse(urlSubmit)
	if err != nil {
		fmt.Printf("URL inválida: %v\n", err)
		return
	}
	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	fmt.Printf("Iniciando prueba de carga contra: %s (Base: %s)\n", urlSubmit, baseURL)
	fmt.Printf("Total de peticiones: %d\n", numRequests)
	fmt.Printf("Nivel de concurrencia: %d\n", concurrency)
	fmt.Println("---------------------------------")

	var wg sync.WaitGroup
	resultsChan := make(chan Result, numRequests)
	taskQueue := make(chan string, numRequests)

	// Lanzar los workers
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker(i, baseURL, taskQueue, resultsChan, &wg)
	}

	totalStart := time.Now()

	// Enviar "trabajos"
	for i := 0; i < numRequests; i++ {
		taskQueue <- urlSubmit
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
			// Imprime el error para que sepamos qué falló (ej. backpressure, timeout)
			fmt.Printf("[Error] %v\n", result.Error)
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
	fmt.Printf("p50 (Mediana): %s\n", p50.Round(time.Millisecond))
	fmt.Printf("p95: %s\n", p95.Round(time.Millisecond))
	fmt.Printf("p99: %s\n", p99.Round(time.Millisecond))
	fmt.Println("---------------------------------")
}