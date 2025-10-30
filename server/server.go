package server

import (
	"P1/jobs"
	"bufio"
	"crypto/rand" // NUEVA IMPORTACIÓN
	"encoding/hex" // NUEVA IMPORTACIÓN
	"fmt"
	"net"
	"strings"
)

type Server struct {
	port int
	Manager *jobs.Manager
}

func NewServer(port int, manager *jobs.Manager) *Server {
	return &Server{port: port, Manager: manager}
}

func (s *Server) Start() {
	addr := fmt.Sprintf(":%d", s.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	fmt.Printf("Servidor escuchando en %s\n", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error al aceptar conexión:", err)
			continue
		}
		
		// Generar Request ID único para esta conexión
		b := make([]byte, 8)
		rand.Read(b)
		reqID := hex.EncodeToString(b)
		
		fmt.Printf("[%s] Nueva conexión desde %s\n", reqID, conn.RemoteAddr())

		// cada conexión se maneja en una goroutine
		go s.handleConnection(conn, reqID) // <-- CAMBIO: Pasar reqID
	}
}

// Maneja una conexión individual.
// Lee la solicitud, procesa y envía la respuesta.
// cuando termina, cierra la conexión.
func (s *Server) handleConnection(conn net.Conn, reqID string) { // <-- CAMBIO: Aceptar reqID
	defer conn.Close()
	reader := bufio.NewReader(conn)

	requestLine, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("[%s] Error al leer solicitud: %v\n", reqID, err)
		return
	}
	method, path, version := parseRequestLine(requestLine)

	// Ignorar headers
	for {
		line, _ := reader.ReadString('\n')
		if line == "\r\n" || line == "\n" {
			break
		}
	}

	fmt.Printf("[%s] %s %s %s\n", reqID, version, method, path)

	statusCode, body := HandleRequest(method, path, s.Manager)

	// Construir y enviar respuesta HTTP/1.0 correcta
	response := buildResponse(statusCode, body, reqID)
	conn.Write([]byte(response))
}

func parseRequestLine(line string) (method, path, version string) {
	parts := strings.Fields(line)
	if len(parts) == 3 {
		return parts[0], parts[1], parts[2]
	}
	return "", "", ""
}