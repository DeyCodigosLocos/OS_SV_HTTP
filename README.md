# Proyecto 1: Servidor HTTP Concurrente

## 1. Visión General de la Arquitectura

El servidor está diseñado bajo una arquitectura modular orientada a servicios, desacoplando la capa de red, la lógica de enrutamiento y la ejecución de tareas. El núcleo del sistema es un **Job Manager** asíncrono que controla la concurrencia mediante **Pools de Workers** específicos para cada tipo de tarea.

se usa este diseño para no bloquear el hilo principal y garantizar la robustez del sistema bajo cargas pesadas mediante mecanismos de encolado.

Este proyecto no utiliza dependencias externas mas allá de la biblioteca estandar de go para apegarse a los requerimientos del proyecto.

## 2. Flujo de Peticiones

El servidor soporta dos modelos de ejecución de tareas, cumpliendo con los requisitos de ejecución directa y asíncrona.

### 2.1. Modo Síncrono

Este modo está diseñado para tareas rápidas o para las rutas de CPU/IO que ofrecen ejecución directa.

- Una petición (ej. GET /pi?digits=100) es recibida por server.go.
- server.go asigna una nueva goroutine para manejar la conexión.
- handler.go identifica la ruta /pi.
- El *handler* invoca **directamente** a la función tasks.PiDigits().
- La goroutine de la conexión **se bloquea** hasta que el cálculo de tasks.PiDigits() finaliza.
- server.go construye y envía la respuesta HTTP con el resultado.

Este modo es susceptible a la sobrecarga del sistema si se realizan muchas peticiones pesadas simultáneamente, ya que no utiliza el sistema de colas.

### 2.2. Modo Asíncrono (Job Manager)

Este es el modo de operación principal y robusto, diseñado para tareas pesadas.

- Una petición (ej. GET /jobs/submit?task=pi&digits=1000) es recibida.
- handler.go identifica la ruta /jobs/submit.
- El *handler* extrae los parámetros y llama a manager.Submit().
- manager.Submit() crea un nuevo Job, lo almacena para persistencia y lo envía a la **cola** específica del WorkerPool de "pi".
- El *handler* responde **inmediatamente** al cliente con un job_id (ej. {"job_id": "...", "status": "queued"}). La conexión del cliente finaliza.
- Independientemente, uno de los *workers* del *pool* de "pi" (ej. 2 *workers* configurados) tomará el trabajo de la cola cuando esté disponible.
- El *worker* ejecuta la tarea tasks.PiDigits(), manejando *timeouts* y resultados.
- Al finalizar, el *worker* actualiza el estado del Job (a "done" o "error") en el *manager*.
- El cliente debe sondear (poll) el endpoint GET /jobs/status o GET /jobs/result para obtener el resultado final.

---

## 3. Módulos Principales

El código fuente está estructurado por responsabilidades en los siguientes paquetes:

- **main.go (Punto de Entrada)**
    Responsable de la inicialización del sistema.
    Configura y lanza el Job Manager.
    **Registra** todas las tareas (CPU/IO), asignando a cada una su lógica de ejecución, número de *workers*, profundidad de cola (queueDepth) y *timeout* de ejecución.
    Inicia el servidor HTTP.

- **server/ (Capa de Red)**
    - **server.go**: Abstrae la lógica del socket TCP (net.Listen, net.Accept). Lanza una nueva goroutine por conexión. Genera IDs de trazabilidad (X-Request-Id).
    - **handler.go**: Actúa como el *router* principal. Utiliza un switch para mapear las rutas HTTP (URLs) a la lógica correspondiente (sea una tarea síncrona o una llamada al *Job Manager*).
    - **response.go**: Utilidad para construir respuestas HTTP/1.0 crudas, asegurando el formato correcto de *headers* y cuerpo.

- **jobs/ (Núcleo de Concurrencia)**
    - **job.go**: Define la estructura de datos Job, incluyendo status, priority, result, etc.
    - **manager.go**: El "cerebro" del sistema. Mantiene el estado de todos los *jobs*. Implementa la lógica de Submit (envío a cola), persistencia en disco (JSON), *backpressure* (rechazo si la cola está llena) y limpieza periódica de trabajos antiguos.
    - **worker_pool.go**: La implementación física del control de concurrencia. Cada *pool* contiene un número fijo de *workers* (goroutines) que consumen trabajos de un canal (chan *Job) específico para su tarea.

- **tasks/ (Lógica de Negocio)**
    - **cpubound.go**: Implementaciones de tareas que uso intensivo de CPU (ej. IsPrime, PiDigits con Chudnovsky, MatrixMul).
    - **iobound.go**: Implementaciones de tareas de uso intensivo de E/S (ej. SortFile con *external merge sort*, WordCount por *streaming*).

---

## 4. Extensibilidad 

Para añadir una nueva tarea asíncrona (ej. "nueva_tarea"):

1.  **Implementar la Lógica:** Añadir la función func NuevaTarea(...) en tasks/cpubound.go o tasks/iobound.go.
2.  **Registrar la Tarea:** En main.go, añadir una nueva llamada jobManager.Register():
    ```go
    jobManager.Register("nueva_tarea",
        // Función wrapper que parsea params y llama a tasks.NuevaTarea
        func(params map[string]string, job *jobs.Job) (any, error) {
            // ... lógica de parseo ...
            return tasks.NuevaTarea(...)
        },
        4,  // workers: 4 workers para esta tarea
        16, // queueDepth: 16 espacios en cola
        30*time.Second, // timeout: 30 segundos
    )
    ```
3.  (Opcional) Añadir una ruta síncrona en server/handler.go si se desea.
