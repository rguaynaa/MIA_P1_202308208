// Archivo: src/components/DiskPanel.jsx
import React, { useState } from 'react'
import { runCommand, getDiskPartitions } from '../api/client.js'
import './shared.css'

// Construye una linea de comando MIA a partir de un mapa de flags,
// omitiendo los que vengan vacios. Centraliza el formato "-flag=valor" que
// usa el parser del backend (ver commands.ParseLine).
function buildLine(cmd, flags) {
  const parts = [cmd]
  for (const [key, val] of Object.entries(flags)) {
    if (val === undefined || val === null || val === '') continue
    if (val === true) {
      parts.push(`-${key}`)
    } else {
      parts.push(`-${key}=${val}`)
    }
  }
  return parts.join(' ')
}

export default function DiskPanel({ onActivity }) {
  // --- MKDISK ---
  const [diskPath, setDiskPath] = useState('')
  const [diskSize, setDiskSize] = useState('10')
  const [diskUnit, setDiskUnit] = useState('M')
  const [diskFit, setDiskFit] = useState('FF')
  const [diskMsg, setDiskMsg] = useState(null)

  // --- FDISK ---
  const [partPath, setPartPath] = useState('')
  const [partName, setPartName] = useState('')
  const [partSize, setPartSize] = useState('5')
  const [partUnit, setPartUnit] = useState('M')
  const [partType, setPartType] = useState('P')
  const [partFit, setPartFit] = useState('WF')
  const [partMsg, setPartMsg] = useState(null)

  // --- MOUNT ---
  const [mountPath, setMountPath] = useState('')
  const [mountName, setMountName] = useState('')
  const [mountMsg, setMountMsg] = useState(null)

  // --- MKFS ---
  const [fsId, setFsId] = useState('')
  const [fsType, setFsType] = useState('full')
  const [fsMsg, setFsMsg] = useState(null)
  // --- Explorar particiones de un disco ---
  const [browsePath, setBrowsePath] = useState('')
  const [partitions, setPartitions] = useState(null)
  const [browseMsg, setBrowseMsg] = useState(null)

  async function handleMkdisk(e) {
    e.preventDefault()
    const line = buildLine('mkdisk', { size: diskSize, unit: diskUnit, fit: diskFit, path: diskPath })
    try {
      const { output } = await runCommand(line)
      setDiskMsg({ ok: true, text: output })
      onActivity?.()
    } catch (err) {
      setDiskMsg({ ok: false, text: err.message })
    }
  }

  async function handleRmdisk() {
    if (!diskPath) {
      setDiskMsg({ ok: false, text: 'Indique la ruta del disco a eliminar.' })
      return
    }
    const line = buildLine('rmdisk', { path: diskPath })
    try {
      const { output } = await runCommand(line)
      setDiskMsg({ ok: true, text: output })
      onActivity?.()
    } catch (err) {
      setDiskMsg({ ok: false, text: err.message })
    }
  }

  async function handleFdisk(e) {
    e.preventDefault()
    const line = buildLine('fdisk', {
      size: partSize,
      unit: partUnit,
      type: partType,
      fit: partFit,
      name: partName,
      path: partPath,
    })
    try {
      const { output } = await runCommand(line)
      setPartMsg({ ok: true, text: output })
      onActivity?.()
    } catch (err) {
      setPartMsg({ ok: false, text: err.message })
    }
  }

  async function handleMount(e) {
    e.preventDefault()
    const line = buildLine('mount', { path: mountPath, name: mountName })
    try {
      const { output } = await runCommand(line)
      setMountMsg({ ok: true, text: output })
      // El backend responde "Particion montada: ID=081A Path=... Name=...".
      // Extraemos el valor de ID= para precargar el campo de MKFS.
      const match = output.match(/ID=(\S+)/)
      if (match) setFsId(match[1])
      onActivity?.()
    } catch (err) {
      setMountMsg({ ok: false, text: err.message })
    }
  }

  async function handleMount(e) {
    e.preventDefault()
    const line = buildLine('mount', { path: mountPath, name: mountName })
    try {
      const { output } = await runCommand(line)
      setMountMsg({ ok: true, text: output })
      // El backend responde "Particion montada: ID=081A Path=... Name=...".
      // Extraemos el valor de ID= para precargar el campo de MKFS.
      const match = output.match(/ID=(\S+)/)
      if (match) setFsId(match[1])
      onActivity?.()
    } catch (err) {
      setMountMsg({ ok: false, text: err.message })
    }
  }

  async function handleMkfs(e) {
    e.preventDefault()
    const line = buildLine('mkfs', { id: fsId, type: fsType })
    try {
      const { output } = await runCommand(line)
      setFsMsg({ ok: true, text: output })
      onActivity?.()
    } catch (err) {
      setFsMsg({ ok: false, text: err.message })
    }
  }

  async function handleBrowse(e) {
    e.preventDefault()
    setBrowseMsg(null)
    setPartitions(null)
    try {
      const list = await getDiskPartitions(browsePath)
      setPartitions(list)
    } catch (err) {
      setBrowseMsg({ ok: false, text: err.message })
    }
  }

  return (
    <div className="disk-panel">
      <div className="card">
        <h3 className="card__title">Crear / eliminar disco</h3>
        <form onSubmit={handleMkdisk}>
          <div className="form-grid">
            <div className="field">
              <label className="field__label">Ruta del disco</label>
              <input
                className="field__input"
                value={diskPath}
                onChange={(e) => setDiskPath(e.target.value)}
                placeholder="/home/usr/Disco1.mia"
              />
            </div>
            <div className="field">
              <label className="field__label">Tamaño</label>
              <input className="field__input" value={diskSize} onChange={(e) => setDiskSize(e.target.value)} />
            </div>
            <div className="field">
              <label className="field__label">Unidad</label>
              <select className="field__select" value={diskUnit} onChange={(e) => setDiskUnit(e.target.value)}>
                <option value="K">K</option>
                <option value="M">M</option>
              </select>
            </div>
            <div className="field">
              <label className="field__label">Fit</label>
              <select className="field__select" value={diskFit} onChange={(e) => setDiskFit(e.target.value)}>
                <option value="FF">First Fit</option>
                <option value="BF">Best Fit</option>
                <option value="WF">Worst Fit</option>
              </select>
            </div>
          </div>
          <div style={{ display: 'flex', gap: 10 }}>
            <button className="btn btn--primary" type="submit">Crear disco (MKDISK)</button>
            <button className="btn btn--danger" type="button" onClick={handleRmdisk}>Eliminar disco (RMDISK)</button>
          </div>
        </form>
        {diskMsg && (
          <div className={`feedback ${diskMsg.ok ? 'feedback--ok' : 'feedback--error'}`}>{diskMsg.text}</div>
        )}
      </div>

      <div className="card">
        <h3 className="card__title">Crear partición (FDISK)</h3>
        <form onSubmit={handleFdisk}>
          <div className="form-grid">
            <div className="field">
              <label className="field__label">Ruta del disco</label>
              <input className="field__input" value={partPath} onChange={(e) => setPartPath(e.target.value)} placeholder="/home/usr/Disco1.mia" />
            </div>
            <div className="field">
              <label className="field__label">Nombre</label>
              <input className="field__input" value={partName} onChange={(e) => setPartName(e.target.value)} placeholder="Part1" />
            </div>
            <div className="field">
              <label className="field__label">Tamaño</label>
              <input className="field__input" value={partSize} onChange={(e) => setPartSize(e.target.value)} />
            </div>
            <div className="field">
              <label className="field__label">Unidad</label>
              <select className="field__select" value={partUnit} onChange={(e) => setPartUnit(e.target.value)}>
                <option value="K">K</option>
                <option value="M">M</option>
              </select>
            </div>
            <div className="field">
              <label className="field__label">Tipo</label>
              <select className="field__select" value={partType} onChange={(e) => setPartType(e.target.value)}>
                <option value="P">Primaria</option>
                <option value="E">Extendida</option>
                <option value="L">Lógica</option>
              </select>
            </div>
            <div className="field">
              <label className="field__label">Fit</label>
              <select className="field__select" value={partFit} onChange={(e) => setPartFit(e.target.value)}>
                <option value="WF">Worst Fit</option>
                <option value="BF">Best Fit</option>
                <option value="FF">First Fit</option>
              </select>
            </div>
          </div>
          <button className="btn btn--primary" type="submit">Crear partición</button>
        </form>
        {partMsg && (
          <div className={`feedback ${partMsg.ok ? 'feedback--ok' : 'feedback--error'}`}>{partMsg.text}</div>
        )}
      </div>

      <div className="card">
        <h3 className="card__title">Montar partición (MOUNT)</h3>
        <form onSubmit={handleMount}>
          <div className="form-grid">
            <div className="field">
              <label className="field__label">Ruta del disco</label>
              <input className="field__input" value={mountPath} onChange={(e) => setMountPath(e.target.value)} placeholder="/home/usr/Disco1.mia" />
            </div>
            <div className="field">
              <label className="field__label">Nombre de partición</label>
              <input className="field__input" value={mountName} onChange={(e) => setMountName(e.target.value)} placeholder="Part1" />
            </div>
          </div>
          <button className="btn btn--primary" type="submit">Montar</button>
        </form>
        {mountMsg && (
          <div className={`feedback ${mountMsg.ok ? 'feedback--ok' : 'feedback--error'}`}>{mountMsg.text}</div>
        )}
      </div>
      <div className="card">
        <h3 className="card__title">Formatear partición (MKFS)</h3>
        <p className="disk-panel__hint">
          Paso obligatorio antes de poder iniciar sesión: crea el superbloque, los bitmaps
          y el usuario <span className="mono">root</span> dentro de la partición montada.
        </p>
        <form onSubmit={handleMkfs}>
          <div className="form-grid">
            <div className="field">
              <label className="field__label">ID de partición montada</label>
              <input className="field__input" value={fsId} onChange={(e) => setFsId(e.target.value)} placeholder="081A" />
            </div>
            <div className="field">
              <label className="field__label">Tipo</label>
              <select className="field__select" value={fsType} onChange={(e) => setFsType(e.target.value)}>
                <option value="full">Full</option>
                <option value="fast">Fast</option>
              </select>
            </div>
          </div>
          <button className="btn btn--primary" type="submit">Formatear</button>
        </form>
        {fsMsg && (
          <div className={`feedback ${fsMsg.ok ? 'feedback--ok' : 'feedback--error'}`}>{fsMsg.text}</div>
        )}
      </div>

      <div className="card">
        <h3 className="card__title">Ver particiones de un disco</h3>
        <form onSubmit={handleBrowse}>
          <div className="form-grid">
            <div className="field">
              <label className="field__label">Ruta del disco</label>
              <input className="field__input" value={browsePath} onChange={(e) => setBrowsePath(e.target.value)} placeholder="/home/usr/Disco1.mia" />
            </div>
          </div>
          <button className="btn" type="submit">Consultar</button>
        </form>
        {browseMsg && <div className="feedback feedback--error">{browseMsg.text}</div>}
        {partitions && (
          <table className="table" style={{ marginTop: 14 }}>
            <thead>
              <tr>
                <th>Nombre</th>
                <th>Tipo</th>
                <th>Fit</th>
                <th>Inicio</th>
                <th>Tamaño (bytes)</th>
              </tr>
            </thead>
            <tbody>
              {partitions.length === 0 && (
                <tr><td colSpan={5} style={{ color: 'var(--text-faint)' }}>Sin particiones</td></tr>
              )}
              {partitions.map((p, i) => (
                <tr key={i}>
                  <td>{p.name}</td>
                  <td>
                    <span className={`tag ${p.type === 'E' ? 'tag--extended' : p.type === 'L' ? 'tag--logical' : 'tag--primary'}`}>
                      {p.type === 'P' ? 'primaria' : p.type === 'E' ? 'extendida' : 'lógica'}
                    </span>
                  </td>
                  <td>{p.fit}</td>
                  <td>{p.start}</td>
                  <td>{p.size}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <style>{`
        .disk-panel {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(360px, 1fr));
          gap: 16px;
        }
          .disk-panel__hint {
          font-size: 12px;
          color: var(--text-faint);
          margin: -6px 0 12px 0;
        }
      `}</style>
    </div>
  )
}
