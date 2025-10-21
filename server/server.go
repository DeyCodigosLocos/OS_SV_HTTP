package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

type Server struct {
	port int
}

func NewServer(port int) *Server {
	return &Server{port: port}
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
		fmt.Printf("Nueva conexión desde %s\n", conn.RemoteAddr())

		// cada conexión se maneja en una goroutine
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	requestLine, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error al leer solicitud:", err)
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

	fmt.Printf("[%s] %s %s\n", version, method, path)

	body := HandleRequest(method, path)

	// Construir y enviar respuesta HTTP/1.0 correcta
	response := buildResponse(200, body)
	conn.Write([]byte(response))
}

func parseRequestLine(line string) (method, path, version string) {
	parts := strings.Fields(line)
	if len(parts) == 3 {
		return parts[0], parts[1], parts[2]
	}
	return "", "", ""
}





