// construcción de respuestas HTTP

package server

import "fmt"
import "strings"

func buildResponse(code int, body string, reqID string) string {
	statusText := map[int]string{
		200: "OK",
		400: "Bad Request",
		404: "Not Found",
		500: "Internal Server Error",
		503: "Service Unavailable", 
	}[code]

	bodyBytes := []byte(body)
	contentLength := len(bodyBytes)

	var b strings.Builder
	fmt.Fprintf(&b, "HTTP/1.0 %d %s\r\n", code, statusText)
	fmt.Fprintf(&b, "Content-Type: application/json\r\n")
	fmt.Fprintf(&b, "Content-Length: %d\r\n", contentLength)
	fmt.Fprintf(&b, "Connection: close\r\n")
	
	fmt.Fprintf(&b, "X-Request-Id: %s\r\n", reqID)
	
	fmt.Fprintf(&b, "\r\n") // línea vacía requerida
	b.Write(bodyBytes)

	return b.String()
		
}