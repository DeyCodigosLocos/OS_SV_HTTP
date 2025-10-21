package tasks

import "strings"
import "os"
import "time"
import "fmt"
import "crypto/sha256"
import "encoding/hex"
import "math/rand"


func ToUpper(s string) string { // convierte una cadena a mayúsculas
	return strings.ToUpper(s)
}


func Reverse(s string) string { // invierte una cadena de texto
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func Fibonacci(n int) int { // calcula el número n de la serie Fibonacci
	if n <= 1 {
		return n
	}
	return Fibonacci(n-1) + Fibonacci(n-2)
}

// funciones de archivos 

func CreateFile(name, content string, repeat int) error { //  crea un archivo con contenido repetido 'repeat' veces
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	defer file.Close()

	for i := 0; i < repeat; i++ {
		_, err := file.WriteString(content + "\n")
		if err != nil {
			return err
		}
	}
	return nil
}


func DeleteFile(name string) error { // elimina un archivo
	return os.Remove(name)
}


// funciones de estado y demás utilidades 


func Status(startTime time.Time, activeTasks int) string { // devuelve información básica del servidor
	uptime := time.Since(startTime).Round(time.Second)
	return fmt.Sprintf(
		"{\"uptime\": \"%v\", \"active_tasks\": %d, \"time\": \"%v\"}",
		uptime, activeTasks, time.Now().Format(time.RFC3339),
	)
}


func Timestamp() string { // devuelve el tiempo actual
	return time.Now().Format(time.RFC3339)
}


func Hash(text string) string { // calcula el hash SHA256 de un texto
	h := sha256.Sum256([]byte(text))
	return hex.EncodeToString(h[:])
}


func RandomNumbers(count, min, max int) []int { // genera una lista de números aleatorios entre min y max
	rand.Seed(time.Now().UnixNano())
	result := make([]int, count)
	for i := 0; i < count; i++ {
		result[i] = rand.Intn(max-min+1) + min
	}
	return result
}

// funciones de simulación 


func Sleep(seconds int) { //pausa la ejecución s segundos
	time.Sleep(time.Duration(seconds) * time.Second)
}


func Simulate(seconds int, taskName string) string { // ejecuta una tarea simulada por X segundos
	fmt.Printf("Simulando tarea '%s' por %d segundos...\n", taskName, seconds)
	time.Sleep(time.Duration(seconds) * time.Second)
	return fmt.Sprintf("Tarea '%s' completada en %d segundos", taskName, seconds)
}


func LoadTest(tasks, sleepSeconds int) string { // lanza 'tasks' tareas concurrentes que duermen X segundos
	start := time.Now()
	done := make(chan bool, tasks)

	for i := 0; i < tasks; i++ {
		go func(id int) {
			time.Sleep(time.Duration(sleepSeconds) * time.Second)
			done <- true
		}(i)
	}

	for i := 0; i < tasks; i++ {
		<-done
	}

	elapsed := time.Since(start).Seconds()
	return fmt.Sprintf("LoadTest completado: %d tareas, %d seg c/u, tiempo total: %.2fs",
		tasks, sleepSeconds, elapsed)
}


// funciones de ayuda 


func Help() string {// devuelve una lista con las rutas disponibles
	return `{
  "endpoints": [
    "/fibonacci?num=N",
    "/createfile?name=filename&content=text&repeat=x",
    "/deletefile?name=filename",
    "/status",
    "/reverse?text=abcdef",
    "/toupper?text=abcd",
    "/random?count=n&min=a&max=b",
    "/timestamp",
    "/hash?text=someinput",
    "/simulate?seconds=s&task=name",
    "/sleep?seconds=s",
    "/loadtest?tasks=n&sleep=x",
    "/help"
  ]
}`
}