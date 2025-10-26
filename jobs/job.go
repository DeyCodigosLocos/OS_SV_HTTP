
package jobs

import "time"

type JobStatus string

// Posibles estados de un trabajo
const (
	StatusQueued    = "queued"
	StatusRunning   = "running"
	StatusDone      = "done"
	StatusError     = "error"
	StatusCanceled  = "canceled"
)

// Job representa un trabajo en la cola
type Job struct {
	ID        string            `json:"job_id"`
	Task      string            `json:"task"`
	Params    map[string]string `json:"params"`
	Status    JobStatus         `json:"status"`
	Progress  int               `json:"progress"`
	Result    any               `json:"result"`
	Error     string            `json:"error,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	StartedAt time.Time         `json:"started_at,omitempty"`
	FinishedAt time.Time        `json:"finished_at,omitempty"`
}