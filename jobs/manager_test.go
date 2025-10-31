package jobs

import (
	"net/url"
	"testing"
	"time"
)

func mockTask(params map[string]string, job *Job) (any, error) {
	time.Sleep(50 * time.Millisecond) 
	job.Progress = 100
	return map[string]any{"n": params["n"]}, nil
}

func TestManager_Submit_Lifecycle(t *testing.T) {
	tmpFile := t.TempDir() + "/testjobs.json"

	manager := NewManager(tmpFile, 1*time.Minute, 1*time.Minute)
	manager.Register("mock", mockTask, 1, 1, 1*time.Second)
	defer manager.Close()

	params := url.Values{}
	params.Set("n", "123")

	jobID, status, err := manager.Submit("mock", params, PrioNormal)
	if err != nil {
		t.Fatalf("Submit (1) devolvió un error inesperado: %v", err)
	}
	if status != StatusQueued {
		t.Errorf("Submit (1) status = %s; se esperaba %s", status, StatusQueued)
	}

	time.Sleep(100 * time.Millisecond)

	job, err := manager.GetStatus(jobID)
	if err != nil {
		t.Fatalf("GetStatus (1) devolvió un error: %v", err)
	}
	if job.Status != StatusDone {
		t.Errorf("Job (1) status = %s; se esperaba %s", job.Status, StatusDone)
	}

	resMap, ok := job.Result.(map[string]any)
	if !ok {
		t.Fatalf("El resultado del Job no es un map[string]any")
	}
	if resMap["n"] != "123" {
		t.Errorf("Resultado del Job = %v; se esperaba %v", resMap["n"], "123")
	}
}

// TestManager_Submit_Backpressure prueba que la cola rechace trabajos
func TestManager_Submit_Backpressure(t *testing.T) {
	manager := NewManager("", 1*time.Minute, 1*time.Minute)
	manager.Register("mock", mockTask, 1, 1, 1*time.Second)
	defer manager.Close()

	params := url.Values{}
	params.Set("n", "1")

	_, _, err := manager.Submit("mock", params, PrioNormal)
	if err != nil {
		t.Fatalf("Submit (1) falló: %v", err)
	}

	_, _, err = manager.Submit("mock", params, PrioNormal)

	if err == nil {
		t.Fatalf("Submit (2) no devolvió error; se esperaba ErrBackpressure")
	}
	if err != ErrBackpressure {
		t.Errorf("Submit (2) err = %v; se esperaba %v", err, ErrBackpressure)
	}
}

// TestManager_Get_Cancel prueba GetStatus, GetResult y Cancel
func TestManager_Get_Cancel(t *testing.T) {
	manager := NewManager("", 1*time.Minute, 1*time.Minute)
	manager.Register("mock", mockTask, 1, 1, 1*time.Second)
	defer manager.Close()

	params := url.Values{}
	params.Set("n", "456")

	_, err := manager.GetStatus("bad-id")
	if err != ErrJobNotFound {
		t.Errorf("GetStatus(bad-id) err = %v; se esperaba %v", err, ErrJobNotFound)
	}
	_, err = manager.GetResult("bad-id")
	if err != ErrJobNotFound {
		t.Errorf("GetResult(bad-id) err = %v; se esperaba %v", err, ErrJobNotFound)
	}

	jobID, _, _ := manager.Submit("mock", params, PrioNormal)
	time.Sleep(100 * time.Millisecond) 

	job, err := manager.GetStatus(jobID)
	if err != nil {
		t.Fatalf("GetStatus(jobID) devolvió un error: %v", err)
	}
	if job.Status != StatusDone {
		t.Errorf("Job status = %s; se esperaba %s", job.Status, StatusDone)
	}

	job, err = manager.GetResult(jobID)
	if err != nil {
		t.Fatalf("GetResult(jobID) devolvió un error: %v", err)
	}
	if resMap, _ := job.Result.(map[string]any); resMap["n"] != "456" {
		t.Errorf("Resultado del Job = %v; se esperaba '456'", resMap["n"])
	}

	// --- Prueba Cancel (Job ya terminado) ---
	_, err = manager.Cancel(jobID)
	if err != ErrNotCancelable {
		t.Errorf("Cancel(job_done) err = %v; se esperaba %v", err, ErrNotCancelable)
	}

	// --- Prueba Cancel (Job en cola) ---
	_, _, _ = manager.Submit("mock", params, PrioNormal) // Llena el worker
	jobID2, _, _ := manager.Submit("mock", params, PrioNormal) // Llena la cola

	status, err := manager.Cancel(jobID2)
	if err != nil {
		t.Fatalf("Cancel(job_queued) devolvió un error: %v", err)
	}
	if status != StatusCanceled {
		t.Errorf("Cancel(job_queued) status = %s; se esperaba %s", status, StatusCanceled)
	}

	time.Sleep(100 * time.Millisecond) 
}