import requests
import time
import json

BASE_URL = "http://localhost:8080"

jobs_to_submit = [
    {"task": "isprime", "n": "999983", "prio": "high"},
    {"task": "factor", "n": "360", "prio": "normal"},
    {"task": "pi", "digits": "100", "prio": "low"},
    {"task": "matrixmul", "n": "200", "seed": "42", "prio": "normal"},
    {"task": "sortfile", "name": "data/big_numbers.txt", "algo": "quick", "prio": "normal"},
    {"task": "wordcount", "file": "data/text.txt", "prio": "low"},
    {"task": "grep", "file": "data/text.txt", "pattern": "101", "prio": "normal"},
    {"task": "compress", "file": "data/text.txt", "prio": "low", "codec": "gzip"},
    {"task": "hashfile", "file": "data/text.txt", "prio": "normal", "algo": "sha256"}
]

submitted_jobs = {}

def parse_json_from_http_response(resp_text):
    """Extrae el JSON después de la línea vacía de cabecera HTTP"""
    try:
        body = resp_text.split("\r\n\r\n", 1)[1]
        return json.loads(body)
    except Exception as e:
        print("⚠️  Error parseando JSON:", e)
        return None

print("--- SUBMIT JOBS ---")
for job in jobs_to_submit:
    try:
        r = requests.get(f"{BASE_URL}/jobs/submit", params=job, timeout=10)
        data = parse_json_from_http_response(r.text)
        if data and "job_id" in data:
            job_id = data["job_id"]
            submitted_jobs[job_id] = job
            print(f"✅ Submitted '{job['task']}' -> job_id={job_id}, status={data.get('status')}")
        else:
            print(f"❌ Error al enviar job {job['task']}: respuesta inválida → {r.text[:80]}")
    except Exception as e:
        print(f"💥 Error al enviar job {job['task']}: {e}")

# ------------------------------------------------------------
print("\n--- POLLING STATUS ---")
# ------------------------------------------------------------
for job_id in submitted_jobs:
    try:
        while True:
            r = requests.get(f"{BASE_URL}/jobs/status", params={"id": job_id}, timeout=10)
            data = parse_json_from_http_response(r.text)
            if not data:
                print(f"⚠️  No se pudo obtener estado de {job_id}")
                break
            status = data.get("status")
            progress = data.get("progress", 0)
            print(f"Job {job_id} ({submitted_jobs[job_id]['task']}): {status}, {progress}%")

            if status in ["done", "error", "canceled"]:
                break
            time.sleep(1)
    except Exception as e:
        print(f"💥 Error al consultar status para job {job_id}: {e}")

# ------------------------------------------------------------
print("\n--- GET RESULTS ---")
# ------------------------------------------------------------
for job_id in submitted_jobs:
    try:
        r = requests.get(f"{BASE_URL}/jobs/result", params={"id": job_id}, timeout=10)
        data = parse_json_from_http_response(r.text)
        print(f"📦 Resultado {job_id} ({submitted_jobs[job_id]['task']}): {json.dumps(data, indent=2)}")
    except Exception as e:
        print(f"💥 Error al obtener resultado para job {job_id}: {e}")

print("\n--- 🧠 TEST COMPLETO ---")
