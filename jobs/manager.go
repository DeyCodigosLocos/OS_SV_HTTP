package jobs

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"
	"os"
)

// Manager gestiona los jobs y sus colas
type Manager struct {
	jobs   map[string]*Job           // jobs activos
	queues map[string]chan *Job      // colas por tipo de tarea
	tasks  map[string]TaskFunc       // funciones registradas
	mu     sync.RWMutex              // mutex de protección
	file   string                    // archivo JSON para persistencia
}

// TaskFunc define la firma de las funciones de job
type TaskFunc func(params map[string]string, job *Job) (any, error)

// NewManager crea un nuevo Job Manager y carga jobs desde un archivo JSON si existe
func NewManager(file string) *Manager {
	m := &Manager{
		jobs:   make(map[string]*Job),
		queues: make(map[string]chan *Job),
		tasks:  make(map[string]TaskFunc),
		file:   file,
	}

	// intentar cargar jobs persistidos
	if _, err := os.Stat(file); err == nil {
		data, err := os.ReadFile(file)
		if err == nil {
			_ = json.Unmarshal(data, &m.jobs)
		}
	}

	return m
}

// Registrar función de tarea
func (m *Manager) RegisterTask(name string, fn TaskFunc, queueSize int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks[name] = fn
	m.queues[name] = make(chan *Job, queueSize)
}

// Encolar trabajo
func (m *Manager) Submit(task string, params url.Values) (string, JobStatus, error) {
	m.mu.RLock()
	fn, ok := m.tasks[task]
	m.mu.RUnlock()
	if !ok {
		return "", "", fmt.Errorf("task %s no registrada", task)
	}

	pp := map[string]string{}
	for k, v := range params {
		if len(v) > 0 {
			pp[k] = v[0]
		}
	}

	job := &Job{
		ID:        genID(),
		Task:      task,
		Params:    pp,
		Status:    StatusQueued,
		Progress:  0,
		CreatedAt: time.Now(),
	}

	m.mu.Lock()
	m.jobs[job.ID] = job
	ch := m.queues[task]
	select {
	case ch <- job:
	default:
		ch <- job // bloqueante si cola llena
	}
	m.mu.Unlock()

	// Ejecutar en goroutine
	go m.runJob(job, fn)

	return job.ID, job.Status, nil
}

// Ejecutar trabajo
func (m *Manager) runJob(job *Job, fn TaskFunc) {
	m.mu.Lock()
	job.Status = StatusRunning
	job.StartedAt = time.Now()
	m.mu.Unlock()

	res, err := fn(job.Params, job)

	m.mu.Lock()
	defer m.mu.Unlock()
	if err != nil {
		job.Status = StatusError
		job.Result = map[string]string{"error": err.Error()}
	} else {
		job.Status = StatusDone
		job.Result = res
	}
	job.FinishedAt = time.Now()
}

// Obtener estado
func (m *Manager) GetStatus(id string) (Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[id]
	if !ok {
		return Job{}, fmt.Errorf("job no encontrado")
	}
	return *job, nil
}

// Obtener resultado (ya serializado a JSON)
func (m *Manager) GetResult(id string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[id]
	if !ok {
		return "", fmt.Errorf("job no encontrado")
	}

	switch job.Status {
	case StatusDone:
		data, err := json.Marshal(job.Result)
		if err != nil {
			return "", fmt.Errorf("error serializando resultado: %v", err)
		}
		return string(data), nil
	case StatusError:
		return fmt.Sprintf(`{"error":"%s"}`, job.Error), nil
	default:
		return fmt.Sprintf(`{"status":"%s"}`, job.Status), nil
	}
}

// Cancelar trabajo
func (m *Manager) Cancel(id string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return "", fmt.Errorf("job no encontrado")
	}
	if job.Status != StatusQueued && job.Status != StatusRunning {
		return "not_cancelable", nil
	}
	job.Status = StatusCanceled
	return "canceled", nil
}

// generar IDs simples
func genID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
