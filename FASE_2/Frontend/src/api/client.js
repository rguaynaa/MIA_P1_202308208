const BASE_URL = 'http://3.142.94.44:8080'

async function request(path, options = {}) {
  const res = await fetch(`${BASE_URL}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  const data = await res.json().catch(() => null)
  if (!res.ok) {
    const message = data?.error || `Error ${res.status}`
    throw new Error(message)
  }
  return data
}

// runCommand ejecuta una sola linea de comando (ej. "mkdisk -size=10
// -unit=M -path=/home/user/disco.mia") y devuelve la salida de consola.
export function runCommand(line) {
  return request('/api/command', {
    method: 'POST',
    body: JSON.stringify({ line }),
  })
}

// runScript ejecuta un script .smia completo (texto plano) linea por linea.
export function runScript(content) {
  return request('/api/script', {
    method: 'POST',
    body: JSON.stringify({ content }),
  })
}

// uploadScript sube un archivo .smia y lo ejecuta en el backend.
export async function uploadScript(file) {
  const form = new FormData()
  form.append('file', file)
  const res = await fetch(`${BASE_URL}/api/script/upload`, {
    method: 'POST',
    body: form,
  })
  const data = await res.json().catch(() => null)
  if (!res.ok) {
    throw new Error(data?.error || `Error ${res.status}`)
  }
  return data
}

// getSession consulta la sesion (LOGIN) activa actualmente en el backend.
export function getSession() {
  return request('/api/session')
}

// getMounted lista las particiones actualmente montadas.
export function getMounted() {
  return request('/api/mounted')
}

// getDiskPartitions lista las particiones (primarias/extendida/logicas) de
// un disco dado su path absoluto en el servidor donde corre el backend.
export function getDiskPartitions(path) {
  return request(`/api/disk/partitions?path=${encodeURIComponent(path)}`)
}

// reportFileUrl construye la URL para mostrar/descargar un reporte (imagen
// o texto) generado previamente por el comando REP.
export function reportFileUrl(path) {
  return `${BASE_URL}/api/reports/file?path=${encodeURIComponent(path)}`
}

export { BASE_URL }
