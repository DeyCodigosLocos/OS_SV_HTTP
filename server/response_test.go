package server

import (
	"strings"
	"testing"
)

func TestBuildResponse(t *testing.T) {
	code := 200
	body := `{"msg":"hola"}`
	reqID := "abc-123"

	response := buildResponse(code, body, reqID)

	// Prueba 1: ¿Contiene el código y el texto correctos?
	expectedLine1 := "HTTP/1.0 200 OK"
	if !strings.Contains(response, expectedLine1) {
		t.Errorf("La respuesta no contiene '%s'. Respuesta: \n%s", expectedLine1, response)
	}

	// Prueba 2: ¿Contiene el Content-Length correcto?
	// El body `{"msg":"hola"}` tiene 14 bytes.
	expectedContentLength := "Content-Length: 14"
	if !strings.Contains(response, expectedContentLength) {
		t.Errorf("La respuesta no contiene '%s'. Respuesta: \n%s", expectedContentLength, response)
	}

	// Prueba 3: ¿Contiene el X-Request-Id?
	expectedReqID := "X-Request-Id: abc-123"
	if !strings.Contains(response, expectedReqID) {
		t.Errorf("La respuesta no contiene '%s'. Respuesta: \n%s", expectedReqID, response)
	}

	// Prueba 4: ¿Contiene el cuerpo?
	if !strings.HasSuffix(response, body) {
		t.Errorf("La respuesta no termina con el body '%s'. Respuesta: \n%s", body, response)
	}
}