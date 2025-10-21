package server

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"P1/tasks"
	"time"
)

// HTTPResponse representa una respuesta HTTP lista para enviar.
type HTTPResponse struct {
	StatusCode int
	StatusText string
	Body       string
	ContentType string
}
func HandleRequest(method, path string) string {
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
