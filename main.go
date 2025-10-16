
package main

import (
	"fmt"
	"P1/server"
)

func main() {
	port := 8080
	fmt.Printf("Iniciando servidor en puerto %d...\n", port)

	srv := server.NewServer(port)
	srv.Start()
}