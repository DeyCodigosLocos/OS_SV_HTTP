// construcción de respuestas HTTP

package server

import "fmt"

func buildResponse(code int, body string) string {
	statusText := map[int]string{
		200: "OK",
		400: "Bad Request",
		404: "Not Found",
		500: "Internal Server Error",
	}[code]

	headers := fmt.Sprintf(
		"HTTP/1.0 %d %s\r\n"+
			"Content-Type: application/json\r\n"+
			"Content-Length: %d\r\n"+
			"Connection: close\r\n\r\n",
		code, statusText, len(body),
	)
	return headers + body
}
