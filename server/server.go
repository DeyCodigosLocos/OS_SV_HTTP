// socket, accept loop, manejo de clientes

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
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	fmt.Printf("Servidor escuchando en puerto %d\n", s.port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error al aceptar conexión:", err)
			continue
		}

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

	method, path, _ := parseRequestLine(requestLine)
	fmt.Printf("Solicitud recibida: %s %s\n", method, path)

	response := HandleRequest(method, path)
	conn.Write([]byte(response))
}

func parseRequestLine(line string) (method, path, version string) {
	parts := strings.Fields(line)
	if len(parts) >= 3 {
		return parts[0], parts[1], parts[2]
	}
	return "", "", ""
}