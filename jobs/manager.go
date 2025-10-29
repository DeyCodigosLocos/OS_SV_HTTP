package jobs

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"sync"
	"time"
)

var (
	ErrTaskNotFound  = errors.New("task no registrada")
	ErrBackpressure  = errors.New("cola llena: backpressure")
	ErrJobNotFound   = errors.New("job no encontrado")
	ErrNotCancelable = errors.New("job no cancelable")
)

// -----------------------------------------------------------------------------
// Configuración por tarea
// -----------------------------------------------------------------------------
type taskConf struct {
	fn         TaskFunc
	timeout    time.Duration
	pool       *WorkerPool
}

// -----------------------------------------------------------------------------
// Manager: maneja jobs, pools, persistencia y limpieza
// -----------------------------------------------------------------------------
type Manager struct {
	mu sync.RWMutex

	tasks map[string]*taskConf
	jobs  map[string]*Job
	pools map[string]*WorkerPool

	file            string
	ttl             time.Duration
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// NewManager inicializa el Manager con persistencia y limpieza periódica
func NewManager(file string, ttl, cleanupInterval time.Duration) *Manager {
	m := &Manager{
		tasks:           make(map[string]*taskConf),
		jobs:            make(map[string]*Job),
		pools:           make(map[string]*WorkerPool),
		file:            file,
		ttl:             ttl,
		cleanupInterval: cleanupInterval,
		stopCleanup:     make(chan struct{}),
	}

	// Cargar jobs persistidos
	if _, err := os.Stat(file); err == nil {
		if data, err := os.ReadFile(file); err == nil && len(data) > 0 {
			_ = json.Unmarshal(data, &m.jobs)
		}
	}

	// Arranca limpieza automática
	go m.cleanupLoop()
	return m
}

// -----------------------------------------------------------------------------
// Registro de tareas y creación de pools
// -----------------------------------------------------------------------------
func (m *Manager) Register(name string, fn TaskFunc, workers, queueDepth int, timeout time.Duration) {
	if workers <= 0 {
		workers = 1
	}
	if queueDepth <= 0 {
		queueDepth = 1
	}

	queue := make(chan *Job, queueDepth)

	pool := NewWorkerPool(name, workers, queue, m)
	pool.Start()

	m.mu.Lock()
	m.tasks[name] = &taskConf{fn: fn, timeout: timeout, pool: pool}
	m.pools[name] = pool
	m.mu.Unlock()
}

// -----------------------------------------------------------------------------
// Envío y ejecución de trabajos
// -----------------------------------------------------------------------------
func (m *Manager) Submit(task string, params url.Values, prio JobPriority) (string, JobStatus, error) {
	m.mu.RLock()
	tc, ok := m.tasks[task]
	m.mu.RUnlock()
	if !ok {
		return "", "", ErrTaskNotFound
	}

	pp := map[string]string{}
	for k, v := range params {
		if len(v) > 0 {
			pp[k] = v[0]
		}
	}

	j := &Job{
		ID:        genID(),
		Task:      task,
		Params:    pp,
		Status:    StatusQueued,
		Priority:  prio,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Encolar sin bloquear
	select {
	case tc.pool.Queue <- j:
	default:
		return "", "", ErrBackpressure
	}

	m.mu.Lock()
	m.jobs[j.ID] = j
	m.mu.Unlock()
	m.persist()

	return j.ID, j.Status, nil
}


// runJob ejecuta una tarea concreta asociada a un Job dentro del manager.
// Se encarga de llamar a la función de tarea, manejar errores, y actualizar el estado.
func (m *Manager) runJob(job *Job) {
	m.mu.RLock()
	tc, ok := m.tasks[job.Task]
	m.mu.RUnlock()
	if !ok {
		m.finishWithError(job.ID, fmt.Errorf("tarea '%s' no registrada", job.Task))
		return
	}

	defer func() {
		if r := recover(); r != nil {
			m.finishWithError(job.ID, fmt.Errorf("panic en tarea '%s': %v", job.Task, r))
		}
	}()

	// Canal para recibir resultado o error
	resultCh := make(chan any, 1)
	errCh := make(chan error, 1)

	// Ejecutar tarea concurrentemente (sin bloquear worker)
	go func() {
		res, err := tc.fn(job.Params, job)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- res
	}()

	timeout := tc.timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	select {
	case res := <-resultCh:
		m.finishWithResult(job.ID, res)
	case err := <-errCh:
		m.finishWithError(job.ID, err)
	case <-time.After(timeout):
		m.finishWithError(job.ID, fmt.Errorf("timeout tras %v", timeout))
	}
}


// RunJob ejecuta el trabajo directamente (usado por los workers)
func (m *Manager) RunJob(j *Job) {
	m.setStatus(j.ID, StatusRunning)

	m.mu.RLock()
	tc, ok := m.tasks[j.Task]
	m.mu.RUnlock()
	if !ok {
		m.finishWithError(j.ID, fmt.Errorf("tarea desconocida"))
		return
	}

	resultCh := make(chan any, 1)
	errCh := make(chan error, 1)

	go func() {
		res, err := tc.fn(j.Params, j)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- res
	}()

	timeout := tc.timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	select {
	case res := <-resultCh:
		m.finishWithResult(j.ID, res)
		m.persist()
	case err := <-errCh:
		m.finishWithError(j.ID, err)
		m.persist()
	case <-time.After(timeout):
		m.finishWithError(j.ID, fmt.Errorf("timeout (%s)", timeout))
		m.persist()
	}
}

// -----------------------------------------------------------------------------
// Utilidades de control de jobs
// -----------------------------------------------------------------------------
func (m *Manager) setStatus(jobID string, st JobStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if j, ok := m.jobs[jobID]; ok {
		j.Status = st
		j.UpdatedAt = time.Now()
	}
}

func (m *Manager) finishWithResult(jobID string, res any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if j, ok := m.jobs[jobID]; ok {
		j.Status = StatusDone
		j.Result = res
		j.Progress = 100
		j.UpdatedAt = time.Now()
	}
}

func (m *Manager) finishWithError(jobID string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if j, ok := m.jobs[jobID]; ok {
		j.Status = StatusError
		j.Error = err.Error()
		j.Progress = 100
		j.UpdatedAt = time.Now()
	}
}

func (m *Manager) GetStatus(jobID string) (*Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	j, ok := m.jobs[jobID]
	if !ok {
		return nil, ErrJobNotFound
	}
	cp := *j
	return &cp, nil
}

func (m *Manager) GetResult(jobID string) (*Job, error) {
	return m.GetStatus(jobID)
}

func (m *Manager) Cancel(jobID string) (JobStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[jobID]
	if !ok {
		return "", ErrJobNotFound
	}
	if j.Status != StatusRunning && j.Status != StatusQueued {
		return "", ErrNotCancelable
	}
	j.Status = StatusCanceled
	j.Progress = 100
	j.UpdatedAt = time.Now()
	m.persist()
	return j.Status, nil
}

// -----------------------------------------------------------------------------
// Persistencia y limpieza
// -----------------------------------------------------------------------------
func (m *Manager) persist() {
	if m.file == "" {
		return
	}
	data, err := json.MarshalIndent(m.jobs, "", "  ")
	if err == nil {
		_ = os.WriteFile(m.file, data, 0644)
	}
}

func (m *Manager) cleanupLoop() {
	t := time.NewTicker(m.cleanupInterval)
	defer t.Stop()
	for {
		select {
		case <-m.stopCleanup:
			return
		case <-t.C:
			m.cleanupOnce()
		}
	}
}

func (m *Manager) cleanupOnce() {
	if m.ttl <= 0 {
		return
	}
	cut := time.Now().Add(-m.ttl)

	m.mu.Lock()
	changed := false
	for id, j := range m.jobs {
		if (j.Status == StatusDone || j.Status == StatusError || j.Status == StatusCanceled) && j.UpdatedAt.Before(cut) {
			delete(m.jobs, id)
			changed = true
		}
	}
	m.mu.Unlock()
	if changed {
		m.persist()
	}
}

// -----------------------------------------------------------------------------
// Métricas globales (para /metrics)
// -----------------------------------------------------------------------------
func (m *Manager) WorkerStats() map[string]any {
	out := make(map[string]any)
	for name, pool := range m.pools {
		out[name] = pool.Stats()
	}
	return out
}

func (m *Manager) QueueSizes() map[string]int {
	out := make(map[string]int)
	for name, pool := range m.pools {
		out[name] = len(pool.Queue)
	}
	return out
}

// JobsSnapshot devuelve un snapshot rápido del mapa de jobs.
func (m *Manager) JobsSnapshot() map[string]*Job {
    m.mu.RLock()
    defer m.mu.RUnlock()
    copy := make(map[string]*Job, len(m.jobs))
    for k, v := range m.jobs {
        copy[k] = v
    }
    return copy
}

// -----------------------------------------------------------------------------
// Shutdown ordenado
// -----------------------------------------------------------------------------
func (m *Manager) Close() {
	close(m.stopCleanup)
	for _, pool := range m.pools {
		pool.Stop()
	}
}

// -----------------------------------------------------------------------------
// Utilidad: generar IDs únicos
// -----------------------------------------------------------------------------
func genID() string {
	return fmt.Sprintf("%d%04d", time.Now().UnixNano(), rand.Intn(10000))
}

// CleanupOnce ejecuta una limpieza manual de los jobs expirados o colgados.
// - Elimina jobs completados, cancelados o con error que superen el TTL.
// - Marca como error los jobs "running" que llevan demasiado tiempo activos.
func (m *Manager) CleanupOnce() {
	if m.ttl <= 0 {
		return
	}

	cutoff := time.Now().Add(-m.ttl)
	m.mu.Lock()
	defer m.mu.Unlock()

	changed := false
	for id, job := range m.jobs {
		switch job.Status {
		case StatusDone, StatusError, StatusCanceled:
			// Si el job terminó y su última actualización es anterior al cutoff, se borra
			if job.UpdatedAt.Before(cutoff) {
				delete(m.jobs, id)
				changed = true
			}

		case StatusRunning:
			// Si un job lleva más del TTL corriendo, se marca como error
			if job.UpdatedAt.Before(cutoff) {
				job.Status = StatusError
				job.Error = "limpieza automática: job colgado (timeout global)"
				job.Progress = 100
				job.UpdatedAt = time.Now()
				changed = true
			}
		}
	}

	if changed {
		m.persist()
		fmt.Printf("[Manager] Limpieza ejecutada, jobs restantes: %d\n", len(m.jobs))
	}
}
