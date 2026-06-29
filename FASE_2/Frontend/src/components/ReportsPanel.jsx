// Archivo: src/components/ReportsPanel.jsx
import React, { useState } from 'react'
import { runCommand, reportFileUrl } from '../api/client.js'
import './shared.css'

const REPORT_TYPES = [
  { value: 'mbr', label: 'MBR' },
  { value: 'disk', label: 'Disco' },
  { value: 'inode', label: 'Inodos' },
  { value: 'block', label: 'Bloques' },
  { value: 'bm_inode', label: 'Bitmap de inodos' },
  { value: 'bm_block', label: 'Bitmap de bloques' },
  { value: 'sb', label: 'Superbloque' },
  { value: 'file', label: 'Archivo' },
  { value: 'ls', label: 'Listado (ls)' },
  { value: 'tree', label: 'Árbol (tree)' },
]

export default function ReportsPanel({ session }) {
  const [id, setId] = useState(session?.id || '')
  const [name, setName] = useState('tree')
  const [path, setPath] = useState('/home/usr/reportes/reporte.png')
  const [pathFileLs, setPathFileLs] = useState('')
  const [msg, setMsg] = useState(null)
  const [imgKey, setImgKey] = useState(0)
  const [showImage, setShowImage] = useState(false)

  const needsExtra = name === 'file' || name === 'ls'
  const isImage = /\.(png|jpg|jpeg)$/i.test(path)

  async function handleGenerate(e) {
    e.preventDefault()
    const parts = [`rep`, `-name=${name}`, `-path=${path}`, `-id=${id}`]
    if (needsExtra && pathFileLs) parts.push(`-path_file_ls=${pathFileLs}`)
    try {
      const { output } = await runCommand(parts.join(' '))
      setMsg({ ok: !output.toLowerCase().includes('error'), text: output })
      setShowImage(isImage)
      setImgKey((k) => k + 1)
    } catch (err) {
      setMsg({ ok: false, text: err.message })
    }
  }

  return (
    <div className="reports-panel">
      <div className="card">
        <h3 className="card__title">Generar reporte (REP)</h3>
        <form onSubmit={handleGenerate}>
          <div className="form-grid">
            <div className="field">
              <label className="field__label">ID de partición montada</label>
              <input className="field__input" value={id} onChange={(e) => setId(e.target.value)} placeholder="081A" />
            </div>
            <div className="field">
              <label className="field__label">Tipo de reporte</label>
              <select className="field__select" value={name} onChange={(e) => setName(e.target.value)}>
                {REPORT_TYPES.map((t) => (
                  <option key={t.value} value={t.value}>{t.label}</option>
                ))}
              </select>
            </div>
            <div className="field">
              <label className="field__label">Ruta de salida</label>
              <input className="field__input" value={path} onChange={(e) => setPath(e.target.value)} />
            </div>
            {needsExtra && (
              <div className="field">
                <label className="field__label">{name === 'file' ? 'Ruta del archivo' : 'Ruta de la carpeta'}</label>
                <input
                  className="field__input"
                  value={pathFileLs}
                  onChange={(e) => setPathFileLs(e.target.value)}
                  placeholder="/home/usr/docs"
                />
              </div>
            )}
          </div>
          <button className="btn btn--primary" type="submit">Generar</button>
        </form>
        {msg && <div className={`feedback ${msg.ok ? 'feedback--ok' : 'feedback--error'}`}>{msg.text}</div>}
      </div>

      {showImage && (
        <div className="card">
          <h3 className="card__title">Vista previa</h3>
          <div className="reports-panel__preview">
            <img key={imgKey} src={`${reportFileUrl(path)}&t=${imgKey}`} alt={`Reporte ${name}`} />
          </div>
        </div>
      )}

      <style>{`
        .reports-panel {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(360px, 1fr));
          gap: 16px;
          align-items: start;
        }
        .reports-panel__preview {
          background: var(--bg);
          border: 1px solid var(--line);
          border-radius: var(--radius);
          padding: 12px;
          overflow: auto;
          max-height: 70vh;
        }
        .reports-panel__preview img {
          max-width: 100%;
          display: block;
        }
      `}</style>
    </div>
  )
}
