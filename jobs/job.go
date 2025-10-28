package jobs

import "time"

type JobStatus string
type JobPriority int

const (
	StatusQueued   JobStatus = "queued"
	StatusRunning  JobStatus = "running"
	StatusDone     JobStatus = "done"
	StatusError    JobStatus = "error"
	StatusCanceled JobStatus = "canceled"

	PrioLow    JobPriority = 0
	PrioNormal JobPriority = 1
	PrioHigh   JobPriority = 2
)

// La firma de tus tareas se mantiene.
type TaskFunc func(params map[string]string, j *Job) (any, error)

type Job struct {
	ID        string            `json:"id"`
	Task      string            `json:"task"`
	Params    map[string]string `json:"params"`
	Status    JobStatus         `json:"status"`
	Priority  JobPriority       `json:"priority"`
	Progress  int               `json:"progress"` // 0..100
	ETAMs     int64             `json:"eta_ms"`
	Result    any               `json:"result"`
	Error     string            `json:"error"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}