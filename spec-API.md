# Especificación de la API del Servidor HTTP

Este documento define los contratos para las rutas (endpoints), formatos de solicitud (request) y respuesta (response) del servidor.

## Módulo de Trabajos (Jobs)

El servidor gestiona tareas de larga duración a través de un sistema de trabajos (jobs).

---

### 1. Enviar un Trabajo

Encola una nueva tarea para su ejecución asíncrona.

- **Endpoint:** `GET /jobs/submit`
- **Parámetros de Query:**
    - `task` (string, requerido): El nombre de la tarea a ejecutar (ej. `isprime`, `sortfile`).
    - `...` (variado): Parámetros específicos de la tarea (ej. `n=97`).
- [cite_start]**Respuesta Exitosa (202 Accepted):** [cite: 57]
    - Descripción: El trabajo fue aceptado y encolado. Se devuelve un ID único para el trabajo.
    - Cuerpo (JSON):
    ```json
    {
      "job_id": "a1b2c3d4-e5f6-7890-1234-567890abcdef",
      "status": "queued"
    }
    ```
- **Respuesta de Error (400 Bad Request):**
    - Descripción: Faltan parámetros o son inválidos.
    - Cuerpo (JSON):
    ```json
    {
      "error": "Parámetro 'task' es requerido."
    }
    ```

---

### 2. Consultar Estado de un Trabajo

Obtiene el estado actual de un trabajo previamente enviado.

- **Endpoint:** `GET /jobs/status`
- **Parámetros de Query:**
    - `id` (string, requerido): El `job_id` devuelto por el endpoint de `submit`.
- [cite_start]**Respuesta Exitosa (200 OK):** [cite: 59]
    - Descripción: Devuelve el estado actual, el progreso y un tiempo estimado de finalización.
    - [cite_start]Cuerpo (JSON): [cite: 61, 62, 63]
    ```json
    {
      "status": "running",//"queued","running","done","error","canceled"
      "progress": 50, // Entero de 0 a 100
      "eta_ms": 15000 // Tiempo estimado restante en milisegundos
    }
    ```
- **Respuesta de Error (404 Not Found):**
    - Descripción: El `job_id` no existe.
    - Cuerpo (JSON):
    ```json
    {
      "error": "Job con ID '...' no encontrado."
    }
    ```

---

### 3. Obtener Resultado de un Trabajo

Obtiene el resultado de un trabajo que ya ha finalizado.

- **Endpoint:** `GET /jobs/result`
- **Parámetros de Query:**
    - `id` (string, requerido): El `job_id`.
- [cite_start]**Respuesta Exitosa (200 OK):** [cite: 64]
    - Descripción: Devuelve el JSON con el resultado de la tarea si esta finalizó correctamente (`status: "done"`).
    - Cuerpo (JSON, ejemplo para `isprime`):
    ```json
    {
      "n": 97,
      "is_prime": true,
      "method": "miller-rabin",
      "elapsed_ms": 12
    }
    ```
- [cite_start]**Respuesta de Error (200 OK con cuerpo de error):** [cite: 64]
    - Descripción: Se devuelve si el trabajo terminó con estado `"error"`.
    - Cuerpo (JSON):
    ```json
    {
        "error": "Timeout excedido después de 60 segundos."
    }
    ```
- **Respuesta de Error (404 Not Found):**
    - Descripción: El `job_id` no existe.
- **Respuesta de Error (409 Conflict):**
    - Descripción: El trabajo aún no ha terminado.
    - Cuerpo (JSON):
    ```json
    {
        "status": "running",
        "message": "El resultado no está disponible todavía."
    }
    ```

---

### 4. Cancelar un Trabajo

Intenta cancelar la ejecución de un trabajo que está en estado `queued` o `running`.

- **Endpoint:** `GET /jobs/cancel`
- **Parámetros de Query:**
    - `id` (string, requerido): El `job_id`.
- [cite_start]**Respuesta Exitosa (200 OK):** [cite: 65]
    - Descripción: El trabajo fue cancelado o ya no era cancelable (porque ya había terminado).
    - Cuerpo (JSON):
    ```json
    {
      "job_id": "a1b2c3d4-e5f6-7890-1234-567890abcdef",
      "status": "canceled" // "not_cancelable" si era 'done' o 'error'
    }
    ```
- **Respuesta de Error (404 Not Found):**
    - Descripción: El `job_id` no existe.



## Módulo de Observabilidad

Estos endpoints proveen información sobre el estado y el rendimiento del servidor.

---

### 5. Obtener Estado del Servidor

Provee una vista general del estado actual del servidor y sus componentes.

-   **Endpoint:** `GET /status`
-   **Parámetros:** Ninguno.
-   **Respuesta Exitosa (200 OK):**
    -   Descripción: Devuelve un objeto JSON con métricas clave del estado del servidor.
    -   Cuerpo (JSON):
    ```json
    {
      "uptime_seconds": 3600,
      "server_pid": 12345,
      "connections_handled": 1520,
      "queues_size": {
        "isprime": 5,
        "sortfile": 2,
        "pi": 0
      },
      "workers": {
        "isprime": [
          { "worker_pid": 12350, "status": "busy" },
          { "worker_pid": 12351, "status": "idle" },
          { "worker_pid": 12352, "status": "idle" }
        ],
        "sortfile": [
          { "worker_pid": 12360, "status": "busy" },
          { "worker_pid": 12361, "status": "busy" }
        ]
      }
    }
    ```

---

### 6. Obtener Métricas de Rendimiento

Provee métricas agregadas sobre el rendimiento histórico del servidor, como los tiempos de ejecución.

-   **Endpoint:** `GET /metrics`
-   **Parámetros:** Ninguno.
-   **Respuesta Exitosa (200 OK):**
    -   Descripción: Devuelve un objeto JSON con métricas de rendimiento como latencias y estado de los workers.
    -   Cuerpo (JSON):
    ```json
    {
      "queues": {
        "isprime": 3,
        "sortfile": 1
      },
      "workers": {
        "isprime": {
          "total": 4,
          "busy": 2
        },
         "sortfile": {
          "total": 2,
          "busy": 2
        }
      },
      "latency_ms": {
          "isprime": {
              "avg_wait": 50.5,
              "avg_execution": 1200.0
          },
          "sortfile": {
              "avg_wait": 250.2,
              "avg_execution": 35000.7
          }
      }
    }
    ```