package jobs

import (
	"fmt"
	"sync/atomic"
	"time"
)

// WorkerPool maneja un conjunto de goroutines (workers)
// que procesan trabajos (*Job) para una tarea específica
// (por ejemplo, "isprime" o "sortfile").
type WorkerPool struct {
	Name     string      // nombre de la tarea (por ejemplo, "isprime")
	Queue    chan *Job   // cola de trabajos pendientes
	Workers  int         // número total de workers
	Active   int64       // número actual de workers ocupados
	Manager  *Manager    // referencia al manager principal
	StopChan chan struct{} // canal para detener el pool
	TotalJobs  int64
    TotalTime  int64 // en nanosegundos
}

// NewWorkerPool crea una nueva instancia del pool
func NewWorkerPool(name string, workers int, queue chan *Job, manager *Manager) *WorkerPool {
	return &WorkerPool{
		Name:     name,
		Queue:    queue,
		Workers:  workers,
		Active:   0,
		Manager:  manager,
		StopChan: make(chan struct{}),
	}
}

// Start inicia todos los workers del pool.
// Cada worker escucha trabajos en su cola asociada y los procesa usando Manager.runJob().
func (p *WorkerPool) Start() {
	for i := 0; i < p.Workers; i++ {
		go p.worker(i)
	}
	fmt.Printf("[WorkerPool:%s] iniciado con %d workers\n", p.Name, p.Workers)
}

// worker ejecuta trabajos tomados del canal Queue.
// Si se cierra el canal StopChan, el worker termina su ejecución.
func (p *WorkerPool) worker(id int) {
	start := time.Now()
	for {
		select {
		case <-p.StopChan:
			fmt.Printf("[WorkerPool:%s] worker %d detenido\n", p.Name, id)
			return
		case job, ok := <-p.Queue:
			if !ok {
				fmt.Printf("[WorkerPool:%s] cola cerrada, worker %d termina\n", p.Name, id)
				return
			}

			atomic.AddInt64(&p.Active, 1)
			start := time.Now()

			p.Manager.setStatus(job.ID, StatusRunning)
			fmt.Printf("[WorkerPool:%s] worker %d procesando job %s\n", p.Name, id, job.ID)

			// Ejecutar el trabajo
			p.Manager.runJob(job)

			elapsed := time.Since(start)
			fmt.Printf("[WorkerPool:%s] worker %d completó job %s en %v\n",
				p.Name, id, job.ID, elapsed)

			atomic.AddInt64(&p.Active, -1)
		}
		elapsed := time.Since(start)
		atomic.AddInt64(&p.TotalJobs, 1)
		atomic.AddInt64(&p.TotalTime, elapsed.Nanoseconds())
	}
	
}

// Stop detiene todos los workers de forma ordenada.
func (p *WorkerPool) Stop() {
	close(p.StopChan)
	close(p.Queue)
	fmt.Printf("[WorkerPool:%s] pool detenido\n", p.Name)
}

// Stats devuelve estadísticas básicas del pool:
// número de workers totales, activos y tamaño de la cola.
func (p *WorkerPool) Stats() map[string]any {
	avg := float64(0)
	if p.TotalJobs > 0 {
		avg = float64(p.TotalTime) / float64(p.TotalJobs) / 1e6
	}
	return map[string]any{
		"workers":  p.Workers,
		"active":   atomic.LoadInt64(&p.Active),
		"queued":   len(p.Queue),
		"capacity": cap(p.Queue),
		"avg_ms":   avg,
	}

}
