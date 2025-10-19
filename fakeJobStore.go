package main

import (
	"crypto/rand" 
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Job representa una tarea que se procesa de forma asíncrona.
type Job struct {
	ID       string      `json:"job_id"`
	Task     string      `json:"task"`
	Status   string      `json:"status"` // "queued", "running", "done", "error", "canceled"
	Progress int         `json:"progress"` // 0-100
	Result   interface{} `json:"result,omitempty"` // omitempty: no incluir si está vacío
	Error    string      `json:"error,omitempty"`
}

// Base de datos simulada
type FakeJobStore struct {
	mu       sync.RWMutex      // Mutex para proteger el acceso al mapa
	jobs     map[string]*Job   // Mapa para guardar jobs por ID
	filePath string            // Archivo para persistencia
}

// Intenta cargar los jobs existentes desde el archivo de persistencia.
func NewFakeJobStore(filePath string) (*FakeJobStore, error) {
	store := &FakeJobStore{
		jobs:     make(map[string]*Job),
		filePath: filePath,
	}

	// Intenta cargar los datos desde el archivo, si existe
	if err := store.loadFromFile(); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No se encontró archivo de persistencia, iniciando uno nuevo.")
		} else {
			return nil, fmt.Errorf("error al cargar job store: %w", err)
		}
	}
	
	fmt.Printf("Job store inicializado. %d jobs cargados.\n", len(store.jobs))
	return store, nil
}

// Carga el estado desde el archivo JSON
// solo se llama desde el cosntructor
func (s *FakeJobStore) loadFromFile() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	// Decodifica el JSON en nuestro mapa de jobs
	if err := json.Unmarshal(data, &s.jobs); err != nil {
		return fmt.Errorf("error al decodificar archivo de store: %w", err)
	}
	return nil
}

// Guarda el estado actual en el archivo JSON.
func (s *FakeJobStore) saveToFile_unsafe() error {
	// Codificamos el mapa de jobs a JSON con indentación
	data, err := json.MarshalIndent(s.jobs, "", "  ")
	if err != nil {
		return fmt.Errorf("error al codificar jobs a JSON: %w", err)
	}

	return os.WriteFile(s.filePath, data, 0644)
}

// Añade un nuevo trabajo al store y lo guarda en disco.
func (s *FakeJobStore) CreateJob(task string) (*Job, error) {
	s.mu.Lock() 
	defer s.mu.Unlock()

	// Generar un ID único usando crypto/rand (16 bytes = 128 bits)
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("error al generar ID de job: %w", err)
	}
	// de bytes a hexadecimal
	jobID := fmt.Sprintf("%x", b)

	job := &Job{
		ID:       jobID, 
		Task:     task,
		Status:   "queued",
		Progress: 0,
	}

	s.jobs[job.ID] = job

	if err := s.saveToFile_unsafe(); err != nil {
		fmt.Printf("ADVERTENCIA: No se pudo persistir el job store: %v\n", err)
	}

	return job, nil
}

// Obtiene un trabajo por ID
func (s *FakeJobStore) GetJob(id string) (*Job, error) {
	s.mu.RLock() 
	defer s.mu.RUnlock()

	job, ok := s.jobs[id]
	if !ok {
		return nil, fmt.Errorf("job con ID '%s' no encontrado", id)
	}
	return job, nil
}

// UpdateJob permite modificar un job.
// Recibe una función "updater" que contiene la lógica de la modificación.
func (s *FakeJobStore) UpdateJob(id string, updater func(job *Job) error) (*Job, error) {
	s.mu.Lock() // Bloqueamos para ESCRITURA
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return nil, fmt.Errorf("job con ID '%s' no encontrado", id)
	}

	// Aplicamos la función de actualización al job
	if err := updater(job); err != nil {
		return nil, fmt.Errorf("falló la actualización del job: %w", err)
	}

	// Guardamos los cambios en el archivo
	if err := s.saveToFile_unsafe(); err != nil {
		fmt.Printf("ADVERTENCIA: No se pudo persistir la actualización del store: %v\n", err)
	}

	return job, nil
}
// --- MOCK WORKER ---

// simula ser un trabajador que procesa tareas.
type MockWorker struct {
	store *FakeJobStore
}

// crea un nuevo worker conectado al store
func NewMockWorker(store *FakeJobStore) *MockWorker {
	return &MockWorker{store: store}
}

// simula la ejecución de un trabajo.
func (w *MockWorker) ProcessJob(job *Job) {
	fmt.Printf("[Worker] Iniciando procesamiento del job %s (tarea: %s)\n", job.ID, job.Task)

	w.store.UpdateJob(job.ID, func(j *Job) error {
		j.Status = "running"
		j.Progress = 0
		return nil
	})

	// 2. Simular la primera mitad del trabajo
	time.Sleep(1 * time.Second)

	// 3. Simular progreso
	w.store.UpdateJob(job.ID, func(j *Job) error {
		j.Progress = 50
		return nil
	})
	fmt.Printf("[Worker] Job %s al 50%%\n", job.ID)

	// 4. Simular la segunda mitad del trabajo
	time.Sleep(1 * time.Second)

	// 5. Simular finalización
	type PrimeResult struct {
		N       int  `json:"n"`
		IsPrime bool `json:"is_prime"`
	}

	w.store.UpdateJob(job.ID, func(j *Job) error {
		j.Status = "done"
		j.Progress = 100
		j.Result = PrimeResult{N: 97, IsPrime: true} 
		return nil
	})

	fmt.Printf("[Worker] Job %s finalizado con éxito.\n", job.ID)
}

/*
// main pra pruebas
func main() {
	// 1. Inicializar el store
	store, err := NewFakeJobStore("jobstore.json")
	if err != nil {
		panic(err)
	}

	// 2. Crear nuestro worker
	worker := NewMockWorker(store)

	// 3. Crear un nuevo job para procesar
	fmt.Println("--- [Cliente] Creando Job para el worker ---")
	jobToProcess, err := store.CreateJob("isprime")
	if err != nil {
		panic(err)
	}
	fmt.Printf("--- [Cliente] Job creado: %s\n", jobToProcess.ID)

	// 4. Entregar el job al worker (en segundo plano)
	go worker.ProcessJob(jobToProcess)

	// 5. Simular la API de /jobs/status
	fmt.Println("\n--- [Cliente] Iniciando sondeo de /jobs/status ---")
	for {
		time.Sleep(750 * time.Millisecond)

		j, _ := store.GetJob(jobToProcess.ID)
		fmt.Printf("[Cliente] Consultando... Job %s: %s (%d%%)\n", j.ID, j.Status, j.Progress)

		if j.Status == "done" || j.Status == "error" {
			break 
		}
	}

	// 6. Simular la API de /jobs/result
	fmt.Println("\n--- [Cliente] Simulando consulta de /jobs/result ---")
	finalJob, _ := store.GetJob(jobToProcess.ID)
	if finalJob.Status == "done" {
		resultJSON, _ := json.Marshal(finalJob.Result)
		fmt.Printf("[Cliente] Resultado Obtenido: %s\n", string(resultJSON))
	} else {
		fmt.Printf("[Cliente] El job no ha terminado o falló: %s\n", finalJob.Status)
	}
}