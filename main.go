package main

import (
	"fmt"
	"strconv"
	"P1/server"
	"P1/tasks"
	"P1/jobs"
	"time"
	"flag"
)

// Instancia global del Job Manager
var jobManager *jobs.Manager

func main() {

	jobManager := jobs.NewManager("jobs_data.json", 10*time.Minute, 30*time.Second)

	// --- Registrar tareas CPU-bound ---
	jobManager.Register("isprime",
		func(p map[string]string, j *jobs.Job) (any, error) {
			// validar parámetro n
			nStr, ok := p["n"]
			if !ok || nStr == "" {
				return map[string]any{"error": "falta parámetro n"}, nil
			}
			n, err := strconv.ParseInt(nStr, 10, 64)
			if err != nil {
				return map[string]any{"n": nStr, "error": "parámetro n inválido (debe ser entero)"}, nil
			}
			if n < 2 {
				return map[string]any{"n": n, "is_prime": false}, nil
			}

			// calcular primalidad
			prime, err := tasks.IsPrime(n)
			if err != nil {
				return map[string]any{"n": n, "error": err.Error()}, nil
			}

			j.Progress = 100
			return map[string]any{"n": n, "is_prime": prime}, nil
		},
		4,              // workers
		64,             // queueDepth
		60*time.Second, // timeout
	)

	jobManager.Register("factor",
		func(params map[string]string, job *jobs.Job) (any, error) {
			nStr, ok := params["n"]
			if !ok || nStr == "" {
				return map[string]any{"error": "falta parámetro n"}, nil
			}
			n, err := strconv.ParseInt(nStr, 10, 64)
			if err != nil {
				return map[string]any{"n": nStr, "error": "n inválido"}, nil
			}

			factors, err := tasks.Factor(n)
			job.Progress = 100
			if err != nil {
				return map[string]any{"n": n, "error": err.Error()}, nil
			}
			return map[string]any{"n": n, "factors": factors}, nil
		},
		3, 16, 60*time.Second)

	jobManager.Register("pi",
		func(params map[string]string, job *jobs.Job) (any, error) {
			digitsStr, ok := params["digits"]
			if !ok {
				return map[string]any{"error": "falta parámetro digits"}, nil
			}
			digits, err := strconv.Atoi(digitsStr)
			if err != nil {
				return map[string]any{"error": "digits inválido"}, nil
			}
			pi, err := tasks.PiDigits(digits)
			job.Progress = 100
			if err != nil {
				return map[string]any{"digits": digits, "error": err.Error()}, nil
			}
			return map[string]any{"digits": digits, "pi": pi}, nil
		},
		2, 8, 90*time.Second)

	jobManager.Register("matrixmul",
		func(params map[string]string, job *jobs.Job) (any, error) {
			sizeStr := params["size"] // <-- CORREGIDO: "size"
			seedStr := params["seed"]

			size, errSize := strconv.Atoi(sizeStr)
			seed, errSeed := strconv.ParseInt(seedStr, 10, 64)

			if errSize != nil || errSeed != nil {
				return nil, fmt.Errorf("parámetros 'size' o 'seed' inválidos")
			}

			hash, err := tasks.MatrixMul(size, seed)
			job.Progress = 100
			if err != nil {
				return nil, err 
			}
			return map[string]any{"size": size, "hash": hash}, nil
		},
		2, 8, 120*time.Second)

	// --- Registrar tareas IO-bound ---
	jobManager.Register("sortfile",
		func(params map[string]string, job *jobs.Job) (any, error) {
			name := params["name"]
			algo := params["algo"]
			start := time.Now()
			out, elapsed, err := tasks.SortFile(name, algo)

			job.Progress = 100
			if err != nil {
				fmt.Printf("[sortfile] Error procesando '%s' con algoritmo '%s': %v\n", name, algo, err)
				return map[string]any{"error": err.Error()}, err
			}

			fmt.Printf("[sortfile] Terminado en %.2fs\n", time.Since(start).Seconds())
			return map[string]any{"output": out, "elapsed_ms": elapsed}, nil
		},
		1, 2, 120*time.Second)

	jobManager.Register("wordcount",
		func(params map[string]string, job *jobs.Job) (any, error) {
			name := params["file"]
			lines, words, bytes, err := tasks.WordCount(name)
			job.Progress = 100
			if err != nil {
				return map[string]any{"error": err.Error()}, nil
			}
			return map[string]any{"lines": lines, "words": words, "bytes": bytes}, nil
		},
		2, 4, 60*time.Second)

	jobManager.Register("grep",
		func(params map[string]string, job *jobs.Job) (any, error) {
			name := params["file"]
			pattern := params["pattern"]
			count, lines, err := tasks.Grep(name, pattern)
			job.Progress = 100
			if err != nil {
				return map[string]any{"error": err.Error()}, nil
			}
			return map[string]any{"count": count, "lines": lines}, nil
		},
		2, 4, 60*time.Second)

	jobManager.Register("compress",
		func(params map[string]string, job *jobs.Job) (any, error) {
			name := params["file"]
			codec := params["codec"]
			out, size, err := tasks.Compress(name, codec)
			job.Progress = 100
			if err != nil {
				return map[string]any{"error": err.Error()}, nil
			}
			return map[string]any{"output": out, "size": size}, nil
		},
		1, 2, 90*time.Second)

	jobManager.Register("hashfile",
		func(params map[string]string, job *jobs.Job) (any, error) {
			name := params["file"]
			hash, err := tasks.HashFile(name)
			job.Progress = 100
			if err != nil {
				return map[string]any{"error": err.Error()}, nil
			}
			return map[string]any{"hash": hash}, nil
		},
		2, 4, 60*time.Second)

	portPtr := flag.Int("port", 8080, "Puerto TCP para escuchar")
	flag.Parse()
	port := *portPtr
	fmt.Printf("Iniciando servidor en puerto %d...\n", port)

	srv := server.NewServer(port, jobManager)
	srv.Start()
}