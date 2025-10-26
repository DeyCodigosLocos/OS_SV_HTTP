package server

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"P1/tasks"
	"time"
	"P1/jobs"
	"encoding/json"
)

// HTTPResponse representa una respuesta HTTP lista para enviar.
type HTTPResponse struct {
	StatusCode int
	StatusText string
	Body       string
	ContentType string
}
func HandleRequest(method, path string, manager *jobs.Manager) string {
	if method != "GET" {
		return buildResponse(400, `{"error": "Método no soportado, use GET"}`)
	}

	// Separar ruta y parámetros
	parts := strings.SplitN(path, "?", 2)
	route := parts[0]
	var params url.Values
	if len(parts) > 1 {
		params, _ = url.ParseQuery(parts[1])
	}

	switch route {

	// --------------------------
	// Tareas básicas
	// --------------------------

	case "/fibonacci":
		n := parseIntParam(params, "num", -1)
		if n < 0 {
			return buildResponse(400, `{"error": "Parámetro num inválido"}`)
		}
		result := tasks.Fibonacci(n)
		body := fmt.Sprintf(`{"n": %d, "result": %d}`, n, result)
		return buildResponse(200, body)

	case "/reverse":
		text := parseStringParam(params, "text", "")
		if text == "" {
			return buildResponse(400, `{"error": "Falta parámetro text"}`)
		}
		reversed := tasks.Reverse(text)
		body := fmt.Sprintf(`{"input": "%s", "result": "%s"}`, text, reversed)
		return buildResponse(200, body)

	case "/toupper":
		text := parseStringParam(params, "text", "")
		if text == "" {
			return buildResponse(400, `{"error": "Falta parámetro text"}`)
		}
		result := tasks.ToUpper(text)
		body := fmt.Sprintf(`{"input": "%s", "result": "%s"}`, text, result)
		return buildResponse(200, body)

	// --------------------------
	// Archivos
	// --------------------------

	case "/createfile":
		name := parseStringParam(params, "name", "")
		content := parseStringParam(params, "content", "")
		repeat := parseIntParam(params, "repeat", 1)

		if name == "" || content == "" {
			return buildResponse(400, `{"error": "Faltan parámetros name o content"}`)
		}

		err := tasks.CreateFile(name, content, repeat)
		if err != nil {
			body := fmt.Sprintf(`{"error": "%s"}`, err.Error())
			return buildResponse(500, body)
		}

		body := fmt.Sprintf(`{"message": "Archivo %s creado correctamente"}`, name)
		return buildResponse(200, body)

	case "/deletefile":
		name := parseStringParam(params, "name", "")
		if name == "" {
			return buildResponse(400, `{"error": "Falta parámetro name"}`)
		}

		err := tasks.DeleteFile(name)
		if err != nil {
			body := fmt.Sprintf(`{"error": "%s"}`, err.Error())
			return buildResponse(500, body)
		}

		body := fmt.Sprintf(`{"message": "Archivo %s eliminado correctamente"}`, name)
		return buildResponse(200, body)

	// --------------------------
	// Estado y utilidades
	// --------------------------

	case "/status":
		body := tasks.Status(time.Now(), 0)
		return buildResponse(200, body)

	case "/timestamp":
		body := fmt.Sprintf(`{"timestamp": "%s"}`, tasks.Timestamp())
		return buildResponse(200, body)

	case "/hash":
		text := parseStringParam(params, "text", "")
		if text == "" {
			return buildResponse(400, `{"error": "Falta parámetro text"}`)
		}
		hash := tasks.Hash(text)
		body := fmt.Sprintf(`{"input": "%s", "hash": "%s"}`, text, hash)
		return buildResponse(200, body)

	case "/random":
		count := parseIntParam(params, "count", 1)
		min := parseIntParam(params, "min", 0)
		max := parseIntParam(params, "max", 100)
		nums := tasks.RandomNumbers(count, min, max)
		body := fmt.Sprintf(`{"count": %d, "min": %d, "max": %d, "numbers": %v}`,
			count, min, max, nums)
		return buildResponse(200, body)

	// --------------------------
	// Simulación / Carga
	// --------------------------

	case "/simulate":
		seconds := parseIntParam(params, "seconds", 1)
		task := parseStringParam(params, "task", "default")
		result := tasks.Simulate(seconds, task)
		body := fmt.Sprintf(`{"task": "%s", "duration": %d, "status": "%s"}`,
			task, seconds, result)
		return buildResponse(200, body)

	case "/sleep":
		seconds := parseIntParam(params, "seconds", 1)
		tasks.Sleep(seconds)
		body := fmt.Sprintf(`{"message": "Sleep de %d segundos completado"}`, seconds)
		return buildResponse(200, body)

	case "/loadtest":
		n := parseIntParam(params, "tasks", 5)
		sleep := parseIntParam(params, "sleep", 1)
		result := tasks.LoadTest(n, sleep)
		body := fmt.Sprintf(`{"message": "%s"}`, result)
		return buildResponse(200, body)

	// --------------------------
	// Ayuda
	// --------------------------

	case "/help":
		body := tasks.Help()
		return buildResponse(200, body)
	// --------------------------
	// CPU BOUND 
	// --------------------------
	case "/isprime":
		n := parseIntParam(params, "n", -1)
		if n < 0 {
			return `{"error": "Parámetro n inválido"}`
		}

		isPrime, err := tasks.IsPrime(int64(n))
		if err != nil {
			return fmt.Sprintf(`{"error": "%v"}`, err)
		}

		return fmt.Sprintf(`{"n": %d, "is_prime": %v}`, n, isPrime)
	
	case "/factor":
		n := parseIntParam(params, "n", -1)
		if n < 2 {
			return `{"error": "Parámetro n inválido"}`
		}
		factors, err := tasks.Factor(int64(n))
		if err != nil {
			return fmt.Sprintf(`{"error": "%v"}`, err)
		}
		return fmt.Sprintf(`{"n": %d, "factors": %v}`, n, factors)
	
	case "/pi":
		digits := parseIntParam(params, "digits", 1000)
		result := tasks.PiDigits(digits)
		return fmt.Sprintf(`{"digits": %d, "pi": "%s"}`, digits, result)

	case "/mandelbrot":
		width := parseIntParam(params, "width", 100)
		height := parseIntParam(params, "height", 100)
		maxIter := parseIntParam(params, "max_iter", 50)
	
		result, err := tasks.Mandelbrot(width, height, maxIter)
		if err != nil {
			return fmt.Sprintf(`{"error": "%v"}`, err)
		}
	
		return fmt.Sprintf(`{"width": %d, "height": %d, "max_iter": %d, "result": %v}`,
			width, height, maxIter, result)
	
	case "/matrixmul":
		size := parseIntParam(params, "size", 100)
		seed := parseIntParam(params, "seed", 42)
	
		hash, err := tasks.MatrixMul(size, int64(seed))
		if err != nil {
			return fmt.Sprintf(`{"error": "%v"}`, err)
		}
	
		return fmt.Sprintf(`{"size": %d, "seed": %d, "hash": "%s"}`, size, seed, hash)	
		

	// --------------------------
	// IO BOUND 
	// --------------------------

	case "/sortfile":
		name := parseStringParam(params, "name", "")
		algo := parseStringParam(params, "algo", "merge")

		if name == "" {
			return `{"error": "Falta parámetro name"}`
		}

		sortedFile, elapsedMs, err := tasks.SortFile(name, algo)
		if err != nil {
			return fmt.Sprintf(`{"error": "%v"}`, err)
		}

		return fmt.Sprintf(`{"file": "%s", "algorithm": "%s", "output": "%s", "duration_ms": %d}`,
			name, algo, sortedFile, elapsedMs)


	case "/wordcount":
		name := parseStringParam(params, "name", "")
		if name == "" {
			return `{"error": "Falta parámetro name"}`
		}

		lines, words, bytes, err := tasks.WordCount(name)
		if err != nil {
			return fmt.Sprintf(`{"error": "%v"}`, err)
		}

		return fmt.Sprintf(`{"file": "%s", "lines": %d, "words": %d, "bytes": %d}`,
			name, lines, words, bytes)

	case "/grep":
		name := parseStringParam(params, "name", "")
		pattern := parseStringParam(params, "pattern", "")
		if name == "" || pattern == "" {
			return `{"error": "Faltan parámetros name o pattern"}`
		}

		count, lines, err := tasks.Grep(name, pattern)
		if err != nil {
			return fmt.Sprintf(`{"error": "%v"}`, err)
		}

		return fmt.Sprintf(`{"file": "%s", "pattern": "%s", "matches": %d, "lines": %v}`,
			name, pattern, count, lines)

	case "/compress":
		name := parseStringParam(params, "name", "")
		codec := parseStringParam(params, "codec", "gzip")

		if name == "" {
			return `{"error": "Falta parámetro name"}`
		}

		output, size, err := tasks.Compress(name, codec)
		if err != nil {
			return fmt.Sprintf(`{"error": "%v"}`, err)
		}

		return fmt.Sprintf(`{"input": "%s", "output": "%s", "size_bytes": %d}`,
			name, output, size)

	case "/hashfile":
		name := parseStringParam(params, "name", "")
		algo := parseStringParam(params, "algo", "sha256")

		if name == "" {
			return `{"error": "Falta parámetro name"}`
		}

		hash, err := tasks.HashFile(name, algo)
		if err != nil {
			return fmt.Sprintf(`{"error": "%v"}`, err)
		}

		return fmt.Sprintf(`{"file": "%s", "algo": "%s", "hash": "%s"}`,
			name, algo, hash)

	// --------------------------
	// JOB MANAGER 
	// --------------------------
	case "/jobs/submit":
		jobID, status, err := manager.Submit(params.Get("task"), params)
		if err != nil {
			return buildResponse(400, fmt.Sprintf(`{"error": "%v"}`, err))
		}
		body := fmt.Sprintf(`{"job_id": "%s", "status": "%s"}`, jobID, status)
		return buildResponse(200, body)

	case "/jobs/status":
		id := parseStringParam(params, "id", "")
		job, err := manager.GetStatus(id)
		if err != nil {
			return buildResponse(404, `{"error":"Job no encontrado"}`)
		}
		body, _ := json.Marshal(job)
		return buildResponse(200, string(body))

	case "/jobs/result":
		id := parseStringParam(params, "id", "")
		result, err := manager.GetResult(id)
		if err != nil {
			return buildResponse(404, `{"error":"Job no encontrado"}`)
		}
		return buildResponse(200, result)

	case "/jobs/cancel":
		id := parseStringParam(params, "id", "")
		status, err := manager.Cancel(id)
		if err != nil {
			return buildResponse(404, `{"error":"Job no encontrado"}`)
		}
		body := fmt.Sprintf(`{"id":"%s","status":"%s"}`, id, status)
		return buildResponse(200, body)


	default:
		return buildResponse(404, `{"error": "Ruta no encontrada"}`)
	}
}


// ---------------------------
// Funciones auxiliares
// ---------------------------

func parseIntParam(params url.Values, key string, def int) int {
	value := params.Get(key)
	if value == "" {
		return def
	}
	num, err := strconv.Atoi(value)
	if err != nil {
		return def
	}
	return num
}

func parseStringParam(params url.Values, key string, def string) string {
	value := params.Get(key)
	if value == "" {
		return def
	}
	return value
}
