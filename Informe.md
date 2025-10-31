
# Informe Científico: Servidor HTTP Concurrente con Gestión de Tareas Asíncronas

**Autor:** Kendall Josué Piedra Navarro y Deyton Alexander Hernandez Cordoba	
**Curso:** Principios de Sistemas Operativos
**Profesor:** Kenneth Obando Rodriguez
**Fecha:** 30 de octubre de 2025

-----

## 1\. Resumen

Este informe detalla el diseño, implementación y evaluación de un servidor HTTP/1.0 concurrente en Go, construido desde cero. El objetivo principal fue gestionar eficientemente tareas de larga duración (CPU-bound e IO-bound) sin bloquear el servidor, utilizando primitivas de concurrencia de sistemas operativos.

La arquitectura implementada se centra en un **Gestor de Trabajos (Job Manager)** asíncrono, que desacopla la recepción de peticiones de su ejecución. Este gestor utiliza **pools de *workers*** (goroutines) por tipo de tarea y **colas de capacidad limitada** para manejar la concurrencia.

Se desarrolló un agente de pruebas de carga (tester) para medir el rendimiento del sistema. Los resultados de las pruebas demuestran que la arquitectura maneja la carga de forma predecible: la latencia escala linealmente una vez que los *pools* de *workers* se saturan, y el mecanismo de *backpressure* previene el colapso del sistema al rechazar peticiones cuando las colas están llenas. El diseño cumple exitosamente con los requisitos de robustez y escalabilidad.

-----

## 2\. Introducción

En los sistemas modernos, los servidores HTTP a menudo deben manejar peticiones que no son instantáneas. Estas pueden ser tareas de cálculo intensivo (CPU-bound), como generar un fractal, o tareas de E/S pesadas (IO-bound), como ordenar un archivo grande en disco.

Un modelo de servidor simple (ej. "hilo por petición") falla en este escenario, ya que un gran número de peticiones concurrentes para tareas largas agotaría rápidamente los recursos del sistema (CPU, memoria, descriptores de archivos), llevando al colapso del servicio.

El objetivo de este proyecto fue implementar un servidor HTTP/1.0 funcional (sin librerías de alto nivel como net/http) capaz de mitigar este problema. La solución se basa en el patrón **Worker Pool**, donde un número fijo de *workers* consume tareas de una cola de trabajos pendientes. Este diseño permite al servidor controlar la concurrencia, encolar el exceso de demanda y aplicar *backpressure* (contrapresión) para mantener la estabilidad.

Este informe presenta el diseño de dicha arquitectura, la estrategia de pruebas empleada para validarla y un análisis crítico de los resultados de rendimiento bajo perfiles de carga de CPU e I/O.

-----

## 3\. Diseño e Implementación

La arquitectura del servidor se desacopla en cuatro componentes principales:

1.  **Capa de Red (server/)**: Utiliza net.Listen y net.Accept para manejar conexiones TCP crudas. Por cada conexión aceptada, genera un ID de trazabilidad (X-Request-Id) y lanza una nueva goroutine (handleConnection). Esta capa es responsable únicamente de la comunicación HTTP/1.0.

2.  **Enrutador (handler.go)**: Actúa como el *router* principal. Analiza la URL de la petición y la dirige a una de dos lógicas:

      * **Rutas Síncronas (ej. /pi):** Llama directamente a la función de la tarea (ej. tasks.PiDigits()) y bloquea la goroutine de la conexión hasta que la tarea finaliza.
      * **Rutas Asíncronas (ej. /jobs/submit):** Actúa como un cliente del Job Manager.

3.  **Gestor de Trabajos (jobs/manager.go)**: Es el "cerebro" del sistema asíncrono.

      * **Registro:** main.go registra cada tipo de tarea (ej. "pi", "matrixmul") en el gestor, especificando el tamaño del *pool* de *workers*, la profundidad de la cola (queueDepth) y el *timeout* de la tarea.
      * **Sumisión:** Al recibir un trabajo (Submit()), el gestor lo añade a la cola del *pool* correspondiente.
      * **Backpressure:** Si la cola está llena (ej. capacidad = 8), el Submit() falla inmediatamente con un error ErrBackpressure, que el *handler* traduce a un error HTTP 400.
      * **Persistencia:** El gestor serializa el estado de todos los trabajos a un archivo JSON (jobstore.json) en cada cambio, permitiendo la recuperación de estado tras un reinicio (Persistencia Efímera).

4.  **Pools de Workers (jobs/worker_pool.go)**: Es el "músculo" del control de concurrencia.

      * Por cada tarea registrada (ej. "matrixmul"), se crea un WorkerPool con un número fijo de goroutines (ej. 2 *workers*).
      * Estos *workers* son consumidores de larga duración que bloquean en un canal (chan *Job).
      * Cuando un trabajo llega al canal, un *worker* lo toma, ejecuta la tarea (gestionando *timeouts*), y al finalizar, vuelve a esperar por más trabajo. Esto garantiza que solo N tareas de un tipo se ejecuten simultáneamente.

-----

## 4\. Estrategia de Pruebas

Para validar el rendimiento del servidor (Requisitos 106, 136), se desarrolló un agente de pruebas de carga (tester) en Go. Este agente es configurable mediante *flags* (-url, -n, -c) y es capaz de simular el flujo asíncrono completo:

1.  Inicia un cronómetro.
2.  Envía la petición a GET /jobs/submit.
3.  Entra en un bucle de sondeo (polling) contra GET /jobs/status cada 200ms.
4.  Al recibir un estado "done" o "error", detiene el cronómetro.
5.  Reporta la latencia total de extremo a extremo (end-to-end).

Se diseñaron tres escenarios de prueba para recolectar los datos para este informe.

-----

## 5\. Resultados

Las pruebas se ejecutaron en el servidor configurado con los *pools* definidos en main.go.

### 5.1. Prueba 1: Validación de Backpressure

Esta prueba verifica que el servidor rechaza peticiones cuando la cola está llena (Req. 68). Se atacó el *endpoint* matrixmul (Capacidad: 2 *workers* + 8 en cola = 10 total) con 12 peticiones concurrentes.

| Concurrencia (c) | Peticiones (n) | Éxitos (200) | Errores (400) | Resultado |
| :--- | :--- | :--- | :--- | :--- |
| 12 | 12 | 10 | 2 | **ÉXITO** |

**Logs de Errores del Agente:**

```
[Error] submit failed (400): cola llena: backpressure
[Error] submit failed (400): cola llena: backpressure
```

### 5.2. Prueba 2: Perfil de Carga CPU-Bound

Se probó el *endpoint* matrixmul?size=500 (Pool: 2 *Workers*, 8 en Cola) bajo tres perfiles de carga.

| Perfil de Carga | Concurrencia (c) | Peticiones (n) | Éxitos | Errores | Throughput (RPS) | p50 (Mediana) | p99 (Peor Caso) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Baja** | 1 | 5 | 5 | 0 | 2.48 | 403ms | 404ms |
| **Media** | 2 | 5 | 5 | 0 | 4.13 | 403ms | 404ms |
| **Alta** | 8 | 8 | 8 | 0 | 5.66 | 1.211s | 1.413s |

### 5.3. Prueba 3: Perfil de Carga IO-Bound

Se probó el *endpoint* sortfile?name=data/big_numbers.txt (Pool: 1 *Worker*, 2 en Cola) bajo dos perfiles de carga.

| Perfil de Carga | Concurrencia (c) | Peticiones (n) | Éxitos | Errores | Throughput (RPS) | p50 (Mediana) | p99 (Peor Caso) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Baja** | 1 | 1 | 1 | 0 | 0.26 | 3.818s | 3.818s |
| **Alta** | 3 | 3 | 3 | 0 | 0.27 | 7.437s | 11.253s |

-----

## 6\. Discusión y Análisis

El análisis de los resultados confirma que el diseño del *Job Manager* y los *Worker Pools* es exitoso y cumple los objetivos de escalabilidad.

**Análisis de Backpressure:**
La Prueba 1 (Backpressure) fue un éxito rotundo. El servidor aceptó 10 trabajos (2 en *workers*, 8 en cola) y rechazó correctamente los trabajos 11 y 12. Esto previene que un pico de tráfico sature la memoria del servidor con trabajos encolados.

**Análisis de CPU-Bound (matrixmul):**

1.  La prueba de **Carga Baja (c=1)** establece nuestra línea base: una tarea matrixmul (size=500) toma aproximadamente **403ms** en ejecutarse.
2.  En **Carga Media (c=2)**, la concurrencia del agente igualó el tamaño del *pool* de *workers* (2). El *throughput* (RPS) se duplicó (2.48 -\> 4.13), mientras que la latencia p99 se mantuvo plana (404ms). Esto demuestra una **escalabilidad horizontal perfecta** mientras la carga sea menor o igual al número de *workers*.
3.  En **Carga Alta (c=8)**, 8 trabajos saturaron el *pool* de 2 *workers*. El servidor no colapsó (Errores: 0). La latencia p99 se disparó a 1.413s. Este incremento es predecible y esperado:
      * 8 trabajos / 2 workers = 4 tandas de ejecución
      * 403ms (tiempo base) * 4 tandas = 1.612s
      * El resultado experimental (1.413s) es consistente con la predicción teórica (1.6s), demostrando que la cola serializó correctamente los trabajos excedentes.

**Análisis de IO-Bound (sortfile):**

1.  La prueba de **Carga Baja (c=1)** establece la línea base: una tarea sortfile (tarea de disco) toma **3.818s**.
2.  En **Carga Alta (c=3)**, 3 trabajos saturaron el *pool* de 1 solo *worker*. La latencia p99 escaló a 11.253s.
      * 3 trabajos / 1 worker = 3 tandas de ejecución
      * 3.818s (tiempo base) * 3 tandas = 11.454s
      * Nuevamente, el resultado experimental (11.253s) coincide casi perfectamente con la predicción teórica (11.4s). Esto confirma que el *pool* de 1 *worker* previno que 3 tareas de IO-Bound compitieran por el disco simultáneamente (lo que se conoce como *disk thrashing*).

-----

## 7\. Conclusiones

La arquitectura de servidor implementada, basada en un *Job Manager* asíncrono y *Worker Pools* por tarea, cumple exitosamente los objetivos del proyecto.

El sistema demostró ser robusto bajo carga, previniendo el colapso mediante *backpressure* y controlando la concurrencia de forma efectiva. Las pruebas de carga arrojaron un escalado de latencia predecible y lineal, que es el comportamiento deseado para un sistema estable.

Los *bugs* iniciales identificados durante el desarrollo (condiciones de carrera en la persistencia, manejo incorrecto de códigos de estado HTTP) fueron descubiertos y corregidos, validando la importancia de una estrategia de pruebas robusta (nuestro agente tester).
