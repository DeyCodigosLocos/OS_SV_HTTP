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

// Configuración por tarea (pool + cola + timeout)
type taskConf struct {
	fn         TaskFunc
	workers    int
	queueDepth int
	timeout    time.Duration
	queue      chan *Job
}

// Manager con mapas, locks, persistencia y limpieza (TTL)
type Manager struct {
	mu sync.RWMutex

	tasks map[string]*taskConf
	jobs  map[string]*Job

	file            string
	ttl             time.Duration
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// NewManager con persistencia + TTL + limpieza periódica
func NewManager(file string, ttl time.Duration, cleanupInterval time.Duration) *Manager {
	m := &Manager{
		tasks:           make(map[string]*taskConf),
		jobs:            make(map[string]*Job),
		file:            file,
		ttl:             ttl,
		cleanupInterval: cleanupInterval,
		stopCleanup:     make(chan struct{}),
	}

	// Cargar jobs persistidos (si existe)
	if _, err := os.Stat(file); err == nil {
		if data, err := os.ReadFile(file); err == nil && len(data) > 0 {
			_ = json.Unmarshal(data, &m.jobs)
		}
	}

	// Arranca limpieza periódica
	go m.cleanupLoop()
	return m
}

// Registrar una tarea con su pool y cola
func (m *Manager) Register(name string, fn TaskFunc, workers, queueDepth int, timeout time.Duration) {
	if workers <= 0 {
		workers = 1
	}
	if queueDepth <= 0 {
		queueDepth = 1
	}
	tc := &taskConf{
		fn:         fn,
		workers:    workers,
		queueDepth: queueDepth,
		timeout:    timeout,
		queue:      make(chan *Job, queueDepth),
	}

	m.mu.Lock()
	m.tasks[name] = tc
	m.mu.Unlock()

	// Lanzar workers
	for i := 0; i < workers; i++ {
		go m.worker(name, tc)
	}
}

// Encolar un trabajo (con prioridad simple; por ahora la cola es FIFO)
func (m *Manager) Submit(task string, params url.Values, prio JobPriority) (string, JobStatus, error) {
	m.mu.RLock()
	tc, ok := m.tasks[task]
	m.mu.RUnlock()
	if !ok {
		return "", "", ErrTaskNotFound
	}

	// aplanar params
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
		Progress:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// backpressure (no bloquear)
	select {
	case tc.queue <- j:
	default:
		return "", "", ErrBackpressure
	}

	m.mu.Lock()
	m.jobs[j.ID] = j
	m.mu.Unlock()
	m.persist()

	return j.ID, j.Status, nil
}

// Worker: ejecuta con timeout "blando": marcamos timeout si excede, aunque la goroutine siga.
// (Sin refactor de firma; no pasamos context a la tarea)
func (m *Manager) worker(taskName string, tc *taskConf) {
	for j := range tc.queue {
		m.setStatus(j.ID, StatusRunning)

		resultCh := make(chan any, 1)
		errCh := make(chan error, 1)

		// Ejecutar tarea en goroutine
		go func(job *Job) {
			res, err := tc.fn(job.Params, job)
			if err != nil {
				errCh <- err
				return
			}
			resultCh <- res
		}(j)

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
			// Nota: la goroutine interna podría seguir corriendo; en el diseño actual
			// no podemos cancelarla sin cambiar la firma de TaskFunc. Aceptable para P1.
		}
	}
}

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

// Cancel "lógico" (marcar cancelado si estaba corriendo). Sin context, no podemos matar la goroutine.
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

// Persistir a disco
func (m *Manager) persist() {
	if m.file == "" {
		return
	}
	data, err := json.MarshalIndent(m.jobs, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(m.file, data, 0644)
}

// Limpieza periódica por TTL
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

// (opcional) para cerrar limpieza si haces shutdown controlado
func (m *Manager) Close() {
	close(m.stopCleanup)
}

// ID simple
func genID() string {
	return fmt.Sprintf("%d%04d", time.Now().UnixNano(), rand.Intn(10000))
}
