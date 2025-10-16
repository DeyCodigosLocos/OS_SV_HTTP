package server

import (
    "fmt"
    "net/url"
    "strconv"
    "strings"
    "P1/tasks" // cambia esto según tu módulo real
)

// --- Funciones auxiliares para extraer parámetros desde el path ---

func parseIntParam(path, key string, defaultValue int) int {
    u, err := url.Parse(path)
    if err != nil {
        return defaultValue
    }
    value := u.Query().Get(key)
    num, err := strconv.Atoi(value)
    if err != nil {
        return defaultValue
    }
    return num
}

func parseStringParam(path, key string) string {
    u, err := url.Parse(path)
    if err != nil {
        return ""
    }
    return u.Query().Get(key)
}

// --- Manejador principal ---

func HandleRequest(method, path string) string {
    if method != "GET" {
        return buildResponse(400, "Método no soportado")
    }

    switch {
    case strings.HasPrefix(path, "/fibonacci"):
        n := parseIntParam(path, "num", 10)
        result := tasks.Fibonacci(n)
        return buildResponse(200, fmt.Sprintf("{\"result\": %d}", result))

    case strings.HasPrefix(path, "/reverse"):
        text := parseStringParam(path, "text")
        reversed := tasks.Reverse(text)
        return buildResponse(200, fmt.Sprintf("{\"result\": \"%s\"}", reversed))

    default:
        return buildResponse(404, "{\"error\": \"Ruta no encontrada\"}")
    }
}
