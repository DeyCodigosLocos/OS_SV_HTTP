
package main

import (
	"fmt"
	"strconv"
	"P1/server"
	"P1/tasks"
	"P1/jobs"
	"time"
)

// Instancia global del Job Manager
var jobManager = jobs.NewManager("jobs_state.json")

func main() {
	// --- Registrar tareas CPU-bound ---
	jobManager.RegisterTask("isprime", func(params map[string]string, job *jobs.Job) (any, error) {
		n, _ := strconv.ParseInt(params["n"], 10, 64)
		prime, err := tasks.IsPrime(n)
		job.Progress = 100
		if err != nil {
			return map[string]any{"n": n, "error": err.Error()}, nil
		}
		return map[string]any{"n": n, "is_prime": prime}, nil
	}, 5)

	jobManager.RegisterTask("factor", func(params map[string]string, job *jobs.Job) (any, error) {
		n, _ := strconv.ParseInt(params["n"], 10, 64)
		factors, err := tasks.Factor(n)
		job.Progress = 100
		if err != nil {
			return map[string]any{"n": n, "error": err.Error()}, nil
		}
		return map[string]any{"n": n, "factors": factors}, nil
	}, 3)

	jobManager.RegisterTask("pi", func(params map[string]string, job *jobs.Job) (any, error) {
		digits, _ := strconv.Atoi(params["digits"])
		pi, err := tasks.PiDigits(digits)
		job.Progress = 100
		if err != nil {
			return map[string]any{"digits": digits, "error": err.Error()}, nil
		}
		return map[string]any{"digits": digits, "pi": pi}, nil
	}, 2)

	jobManager.RegisterTask("matrixmul", func(params map[string]string, job *jobs.Job) (any, error) {
		size, _ := strconv.Atoi(params["n"])
		seed, _ := strconv.ParseInt(params["seed"], 10, 64)
		hash, err := tasks.MatrixMul(size, seed)
		job.Progress = 100
		if err != nil {
			return map[string]any{"size": size, "error": err.Error()}, nil
		}
		return map[string]any{"size": size, "hash": hash}, nil
	}, 2)

	// --- Registrar tareas IO-bound ---
	jobManager.RegisterTask("sortfile", func(params map[string]string, job *jobs.Job) (any, error) {
		name := params["name"]
		algo := params["algo"]

		start := time.Now()
		out, elapsed, err := tasks.SortFile(name, algo)

		if err != nil {
			fmt.Printf("[sortfile] Error procesando '%s' con algoritmo '%s': %v\n", name, algo, err)
			job.Progress = 100
			return map[string]any{"error": err.Error()}, err
		}

		job.Progress = 100
		fmt.Printf("[sortfile] Terminado en %.2fs\n", time.Since(start).Seconds())
		return map[string]any{"output": out, "elapsed_ms": elapsed}, nil
	}, 1)


	jobManager.RegisterTask("wordcount", func(params map[string]string, job *jobs.Job) (any, error) {
		name := params["file"]
		lines, words, bytes, err := tasks.WordCount(name)
		job.Progress = 100
		if err != nil {
			return map[string]any{"error": err.Error()}, nil
		}
		return map[string]any{"lines": lines, "words": words, "bytes": bytes}, nil
	}, 2)

	jobManager.RegisterTask("grep", func(params map[string]string, job *jobs.Job) (any, error) {
		name := params["file"]
		pattern := params["pattern"]
		count, lines, err := tasks.Grep(name, pattern)
		job.Progress = 100
		if err != nil {
			return map[string]any{"error": err.Error()}, nil
		}
		return map[string]any{"count": count, "lines": lines}, nil
	}, 2)

	jobManager.RegisterTask("compress", func(params map[string]string, job *jobs.Job) (any, error) {
		name := params["file"]
		codec := params["codec"]
		out, size, err := tasks.Compress(name, codec)
		job.Progress = 100
		if err != nil {
			return map[string]any{"error": err.Error()}, nil
		}
		return map[string]any{"output": out, "size": size}, nil
	}, 1)

	jobManager.RegisterTask("hashfile", func(params map[string]string, job *jobs.Job) (any, error) {
		name := params["file"]
		hash, err := tasks.HashFile(name)
		job.Progress = 100
		if err != nil {
			return map[string]any{"error": err.Error()}, nil
		}
		return map[string]any{"hash": hash}, nil
	}, 2)

	
	port := 8080
	fmt.Printf("Iniciando servidor en puerto %d...\n", port)

	srv := server.NewServer(port, jobManager)
	srv.Start()
}