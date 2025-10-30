
# Manual de Usuario: Servidor HTTP y Agente de Pruebas

Este documento describe la ejecución y el uso del servidor HTTP, la estructura de sus *endpoints* (direcciones URL) y el funcionamiento del cliente de pruebas de carga (tester).

---

## 1. Ejecución del Servidor

El servidor se inicia ejecutando el archivo main.go desde la raíz del proyecto.

```bash
go run main.go
```

Por defecto, el servidor se iniciará en el puerto 8080.

### Configuración del Puerto

Para especificar un puerto de escucha diferente, se debe utilizar el *flag* (parámetro) -port al momento de la ejecución.

**Ejemplo (ejecución en el puerto 9090):**

```bash
go run main.go -port=9090
```

Toda la comunicación con el servidor se realizará a través del puerto configurado (ej. http://localhost:9090).

-----

## 2\. Estructura de la API (Endpoints)

La dirección de un *endpoint* se deduce combinando la dirección base del servidor (ej. http://localhost:9080) con la ruta de la tarea específica.

Todos los *endpoints* utilizan el método GET, y los parámetros se proporcionan como *query parameters* en la URL (ej. ?clave=valor&clave2=valor2).

### 2.1. Endpoints Síncronos

Estas rutas ejecutan la tarea de forma directa y bloquean la conexión hasta que el resultado está listo.

**Formato:** `http://[host]:[port]/[nombre-de-ruta]?[parametros]`

**Ejemplos de rutas y parámetros:**

  * /isprime?n=997
  * /pi?digits=1000
  * /matrixmul?size=500&seed=42

### 2.2. Endpoints Asíncronos (Sistema de Trabajos)

Este es el método robusto para tareas pesadas. La ejecución se realiza en segundo plano y no bloquea la conexión.

**Formato (/jobs/submit):**
`http://[host]:[port]/jobs/submit?task=[nombre-de-tarea]&[parametros]`

El parámetro task es obligatorio y debe coincidir con el nombre de la ruta síncrona (ej. pi, matrixmul). Los demás parámetros son idénticos a los usados en la ruta síncrona.

**Ejemplos:**

  * .../jobs/submit?task=pi&digits=1000&prio=high
  * .../jobs/submit?task=matrixmul&size=500&seed=42

### 2.3. Parámetros de Archivo (Tareas IO-Bound)

Para las tareas que operan sobre archivos (como sortfile, wordcount, grep), se utilizan parámetros específicos para indicar la ruta del archivo *en el servidor*.

  * **name** o **file**: Especifica la ruta al archivo (ej. data/big_numbers.txt).
  * **algo**: Usado por sortfile (ej. merge).
  * **pattern**: Usado por grep (ej. 101).

**Ejemplo (Síncrono):**
.../sortfile?name=data/big_numbers.txt&algo=merge

**Ejemplo (Asíncrono):**
.../jobs/submit?task=sortfile&name=data/big_numbers.txt&algo=merge

-----

## 3\. Uso del Cliente de Pruebas (Tester)

El proyecto incluye un agente de pruebas de carga (tester/main.go) para generar métricas de rendimiento (p50, p99, RPS).

### 3.1. Ejecución

El *tester* se ejecuta desde la raíz del proyecto (go run tester/main.go) y se configura obligatoriamente mediante *flags* (parámetros) de línea de comandos.

**Formato del Comando:**

```bash
go run tester/main.go -url="[URL_OBJETIVO]" -n=[PETICIONES] -c=[CONCURRENCIA]
```

### 3.2. Flags de Configuración

  * -url (Obligatorio): La URL completa del *endpoint* que se desea probar.
  * -n: El número **total** de peticiones a realizar (ej. -n=100).
  * -c: El nivel de **concurrencia** (usuarios virtuales simultáneos, ej. -c=20).

### 3.3. Ejemplos de Pruebas

**Prueba Síncrona (Directa):**
Prueba el *endpoint* /matrixmul (síncrono) con 10 usuarios concurrentes, 50 peticiones en total.

```bash
go run tester/main.go -url="http://localhost:9080/matrixmul?size=500&seed=42" -n=50 -c=10
```

**Prueba Asíncrona (Job Manager):**
Prueba el *endpoint* /jobs/submit (asíncrono) para la misma tarea. El *tester* (V4) manejará automáticamente el sondeo (polling) de /jobs/status.

```bash
go run tester/main.go -url="http://localhost:9080/jobs/submit?task=matrixmul&size=500&seed=42" -n=50 -c=10
```
