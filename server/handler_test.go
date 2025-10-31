package server

import (
	"P1/jobs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type mockManager struct {
	jobs.ManagerInterface
}

func (m *mockManager) Submit(task string, params url.Values, prio jobs.JobPriority) (string, jobs.JobStatus, error) {
	if task == "pi" {
		return "job-123", jobs.StatusQueued, nil
	}
	return "", "", jobs.ErrTaskNotFound
}

func (m *mockManager) GetStatus(jobID string) (*jobs.Job, error) {
	if jobID == "job-123" {
		return &jobs.Job{ID: "job-123", Status: jobs.StatusDone, Result: "simulated_result"}, nil
	}
	return nil, jobs.ErrJobNotFound
}

func (m *mockManager) GetResult(jobID string) (*jobs.Job, error) {
	return m.GetStatus(jobID)
}

func (m *mockManager) Cancel(jobID string) (jobs.JobStatus, error) {
	if jobID == "job-123" {
		return jobs.StatusCanceled, nil
	}
	return "", jobs.ErrJobNotFound 
}

func (m *mockManager) WorkerStats() map[string]any                { return nil }
func (m *mockManager) QueueSizes() map[string]int                 { return nil }
func (m *mockManager) JobsSnapshot() map[string]*jobs.Job          { return nil }
func (m *mockManager) CleanupOnce()                               {}
func (m *mockManager) Close()                                     {}
func (m *mockManager) Register(name string, task jobs.TaskFunc, workers int, queueDepth int, timeout time.Duration) {}


func createTempFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "testfile.txt")
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("No se pudo crear el archivo temporal: %v", err)
	}
	return path
}


func TestHandleRequest_Sync(t *testing.T) {
	mockMgr := &mockManager{}

	code, body := HandleRequest("GET", "/status", mockMgr)
	if code != 200 {
		t.Errorf("/status code = %d; se esperaba 200", code)
	}
	if !strings.Contains(body, "uptime") {
		t.Errorf("/status body = %s; se esperaba JSON de status", body)
	}

	code, body = HandleRequest("GET", "/ruta-que-no-existe", mockMgr)
	if code != 404 {
		t.Errorf("/ruta-que-no-existe code = %d; se esperaba 404", code)
	}

	code, body = HandleRequest("POST", "/status", mockMgr)
	if code != 400 {
		t.Errorf("POST /status code = %d; se esperaba 400", code)
	}

	code, body = HandleRequest("GET", "/help", mockMgr)
	if code != 200 {
		t.Errorf("/help code = %d; se esperaba 200", code)
	}
}
// TestHandleRequest_Jobs prueba las rutas del Job Manager
func TestHandleRequest_Jobs(t *testing.T) {
	mockMgr := &mockManager{}

	code, body := HandleRequest("GET", "/jobs/submit?task=pi&digits=100", mockMgr)
	if code != 200 {
		t.Errorf("/jobs/submit code = %d; se esperaba 200. Body: %s", code, body)
	}
	if !strings.Contains(body, "job-123") {
		t.Errorf("/jobs/submit body = %s; se esperaba 'job-123'", body)
	}

	code, _ = HandleRequest("GET", "/jobs/submit", mockMgr)
	if code != 400 {
		t.Errorf("/jobs/submit (sin task) code = %d; se esperaba 400", code)
	}

	code, body = HandleRequest("GET", "/jobs/status?id=job-123", mockMgr)
	if code != 200 {
		t.Errorf("/jobs/status code = %d; se esperaba 200", code)
	}
	if !strings.Contains(body, "simulated_result") {
		t.Errorf("/jobs/status body = %s; se esperaba 'simulated_result'", body)
	}

	code, _ = HandleRequest("GET", "/jobs/status?id=bad-id", mockMgr)
	if code != 404 {
		t.Errorf("/jobs/status (bad-id) code = %d; se esperaba 404", code)
	}

	code, _ = HandleRequest("GET", "/jobs/result?id=bad-id", mockMgr)
	if code != 404 {
		t.Errorf("/jobs/result (bad-id) code = %d; se esperaba 404", code)
	}

	// La prueba ahora espera 404, porque el mock devuelve ErrJobNotFound
	code, _ = HandleRequest("GET", "/jobs/cancel?id=bad-id", mockMgr)
	if code != 404 {
		t.Errorf("/jobs/cancel (bad-id) code = %d; se esperaba 404", code)
	}

	code, _ = HandleRequest("GET", "/metrics", mockMgr)
	if code != 200 {
		t.Errorf("/metrics code = %d; se esperaba 200", code)
	}
}

// TestHandleRequest_Sync_CPU prueba las rutas de CPU síncronas
func TestHandleRequest_Sync_CPU(t *testing.T) {
	var mockMgr *mockManager = nil

	// Caso 1: /isprime (éxito)
	code, body := HandleRequest("GET", "/isprime?n=7", mockMgr)
	if code != 200 {
		t.Errorf("/isprime code = %d; se esperaba 200", code)
	}
	if !strings.Contains(body, `"is_prime": true`) {
		t.Errorf("/isprime body = %s; se esperaba 'is_prime': true", body)
	}

	// Caso 2: /isprime (parámetro inválido)
	code, body = HandleRequest("GET", "/isprime?n=-1", mockMgr)
	if code != 400 {
		t.Errorf("/isprime(bad) code = %d; se esperaba 400", code)
	}

	// Caso 3: /factor (éxito)
	code, body = HandleRequest("GET", "/factor?n=360", mockMgr)
	if code != 200 {
		t.Errorf("/factor code = %d; se esperaba 200", code)
	}
	// El Sprintf de Go añade espacios
	if !strings.Contains(body, `"factors": [[2 3] [3 2] [5 1]]`) {
		t.Errorf("/factor body = %s; se esperaba 'factors': [[2 3] [3 2] [5 1]]", body)
	}

	// Caso 4: /pi (éxito)
	code, body = HandleRequest("GET", "/pi?digits=7", mockMgr)
	if code != 200 {
		t.Errorf("/pi code = %d; se esperaba 200", code)
	}
	// El Sprintf de Go añade espacios y redondea
	if !strings.Contains(body, `"pi": "3.1415927`) {
		t.Errorf("/pi body = %s; se esperaba 'pi': '3.1415927...'", body)
	}
}

// TestHandleRequest_Sync_IO prueba las rutas de I/O síncronas
func TestHandleRequest_Sync_IO(t *testing.T) {
	var mockMgr *mockManager = nil

	path := createTempFile(t, "hola mundo\nadios\n")

	// Caso 1: /wordcount (éxito)
	url := "/wordcount?name=" + path
	code, body := HandleRequest("GET", url, mockMgr)
	if code != 200 {
		t.Errorf("/wordcount code = %d; se esperaba 200. Body: %s", code, body)
	}
	if !strings.Contains(body, `"words": 3`) {
		t.Errorf("/wordcount body = %s; se esperaba 'words': 3", body)
	}

	// Caso 2: /grep (éxito)
	url = "/grep?name=" + path + "&pattern=hola"
	code, body = HandleRequest("GET", url, mockMgr)
	if code != 200 {
		t.Errorf("/grep code = %d; se esperaba 200. Body: %s", code, body)
	}
	if !strings.Contains(body, `"matches": 1`) {
		t.Errorf("/grep body = %s; se esperaba 'matches': 1", body)
	}

	// Caso 3: /grep (parámetros faltantes)
	url = "/grep?name=" + path
	code, _ = HandleRequest("GET", url, mockMgr)
	if code != 400 {
		t.Errorf("/grep (sin pattern) code = %d; se esperaba 400", code)
	}
}

func TestParseRequestLine(t *testing.T) {
	line := "GET /status HTTP/1.0\r\n"
	method, path, version := parseRequestLine(line)

	if method != "GET" {
		t.Errorf("method = %s; se esperaba GET", method)
	}
	if path != "/status" {
		t.Errorf("path = %s; se esperaba /status", path)
	}
	if version != "HTTP/1.0" {
		t.Errorf("version = %s; se esperaba HTTP/1.0", version)
	}
}

func TestParseIntParam(t *testing.T) {
	params := url.Values{}
	params.Set("num", "123")
	params.Set("bad", "abc")

	val := parseIntParam(params, "num", 0)
	if val != 123 {
		t.Errorf("parseIntParam(num) = %d; se esperaba 123", val)
	}
	val = parseIntParam(params, "bad", -1)
	if val != -1 {
		t.Errorf("parseIntParam(bad) = %d; se esperaba -1", val)
	}
	val = parseIntParam(params, "missing", -1)
	if val != -1 {
		t.Errorf("parseIntParam(missing) = %d; se esperaba -1", val)
	}
}

func TestParseStringParam(t *testing.T) {
	params := url.Values{}
	params.Set("text", "hola")

	val := parseStringParam(params, "text", "default")
	if val != "hola" {
		t.Errorf("parseStringParam(text) = %s; se esperaba 'hola'", val)
	}
	val = parseStringParam(params, "missing", "default")
	if val != "default" {
		t.Errorf("parseStringParam(missing) = %s; se esperaba 'default'", val)
	}
}