package main

import (
	"fmt"      
	"io"
	"net/http" // se utiliza solo en el tester para poder hacer requests HTTP al servidor
	"time"
)

func main() {
	url := "http://localhost:8080/status"
	fmt.Printf("lanzando un solo request a %s\n", url)

	start := time.Now()

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error al conectar con el servidor: %v\n", err)
		return
	}
	defer resp.Body.Close() // para ejecutar al finalizar la función

	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		fmt.Printf("Error al leer la respuesta: %v\n", err)
		return
	}
	elapsed := time.Since(start)

	fmt.Printf("Petición completada con código de estado: %d\n", resp.StatusCode)
	fmt.Printf("latencia medida de la petición: %s\n", elapsed)
}