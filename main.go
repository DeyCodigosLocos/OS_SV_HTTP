
package main

import (
	"fmt"
	"strconv"
	"P1/server"
	"P1/tasks"
	"P1/jobs"
)

// Instancia global del Job Manager
var jobManager = jobs.NewManager("jobs_state.json")

func main() {
	// --- Registrar tareas CPU-bound ---
	jobManager.RegisterTask("isprime", func(params map[string]string, job *jobs.Job) (any, error) {
		n, _ := strconv.ParseInt(params["n"], 10, 64)
		prime, _ := tasks.IsPrime(n)
		job.Progress = 100
		return map[string]any{"n": n, "is_prime": prime}, nil
	}, 5) // 5 tareas concurrentes permitidas

	jobManager.RegisterTask("factor", func(params map[string]string, job *jobs.Job) (any, error) {
		n, _ := strconv.ParseInt(params["n"], 10, 64)
		factors, err := tasks.Factor(n)
		if err != nil {
			return nil, err
		}
		job.Progress = 100
		return map[string]any{"n": n, "factors": factors}, nil
	}, 3)

	jobManager.RegisterTask("pi", func(params map[string]string, job *jobs.Job) (any, error) {
		digits, _ := strconv.Atoi(params["digits"])
		pi := tasks.PiDigits(digits)
		job.Progress = 100
		return map[string]any{"digits": digits, "pi": pi}, nil
	}, 2)

	jobManager.RegisterTask("matrixmul", func(params map[string]string, job *jobs.Job) (any, error) {
		size, _ := strconv.Atoi(params["size"])
		seed, _ := strconv.ParseInt(params["seed"], 10, 64)
		hash, err := tasks.MatrixMul(size, seed)
		job.Progress = 100
		return map[string]any{"size": size, "hash": hash}, err
	}, 2)

	// --- Registrar tareas IO-bound ---
	jobManager.RegisterTask("sortfile", func(params map[string]string, job *jobs.Job) (any, error) {
		name := params["name"]
		algo := params["algo"]
		out, elapsed, err := tasks.SortFile(name, algo)
		job.Progress = 100
		return map[string]any{"output": out, "elapsed_ms": elapsed}, err
	}, 1)

	jobManager.RegisterTask("wordcount", func(params map[string]string, job *jobs.Job) (any, error) {
		name := params["name"]
		lines, words, bytes, err := tasks.WordCount(name)
		job.Progress = 100
		return map[string]any{"lines": lines, "words": words, "bytes": bytes}, err
	}, 2)

	jobManager.RegisterTask("grep", func(params map[string]string, job *jobs.Job) (any, error) {
		name := params["name"]
		pattern := params["pattern"]
		count, lines, err := tasks.Grep(name, pattern)
		job.Progress = 100
		return map[string]any{"count": count, "lines": lines}, err
	}, 2)

	jobManager.RegisterTask("compress", func(params map[string]string, job *jobs.Job) (any, error) {
		name := params["name"]
		codec := params["codec"]
		out, size, err := tasks.Compress(name, codec)
		job.Progress = 100
		return map[string]any{"output": out, "size": size}, err
	}, 1)

	jobManager.RegisterTask("hashfile", func(params map[string]string, job *jobs.Job) (any, error) {
		name := params["name"]
		algo := params["algo"]
		hash, err := tasks.HashFile(name, algo)
		job.Progress = 100
		return map[string]any{"hash": hash}, err
	}, 2)


	
	port := 8080
	fmt.Printf("Iniciando servidor en puerto %d...\n", port)

	srv := server.NewServer(port, jobManager)
	srv.Start()
}