// Archivo: src/components/FileSystemPanel.jsx
import React, { useState } from 'react'
import { runCommand } from '../api/client.js'
import './shared.css'

function buildLine(cmd, flags) {
  const parts = [cmd]
  for (const [key, val] of Object.entries(flags)) {
    if (val === undefined || val === null || val === '') continue
    if (val === true) parts.push(`-${key}`)
    else parts.push(`-${key}=${val}`)
  }
  return parts.join(' ')
}

// Una sola fila reutilizable: etiqueta + input + boton, usada para los
// comandos mas simples (RM, CAT, RENAME) para no repetir markup.
function ActionRow({ label, placeholder, secondPlaceholder, twoFields, onRun }) {
  const [value, setValue] = useState('')
  const [second, setSecond] = useState('')
  const [msg, setMsg] = useState(null)

  async function handleClick() {
    try {
      const { output } = await onRun(value, second)
      setMsg({ ok: true, text: output })
    } catch (err) {
      setMsg({ ok: false, text: err.message })
    }
  }

  return (
    <div className="action-row">
      <div className="action-row__fields">
        <input
          className="field__input"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder={placeholder}
        />
        {twoFields && (
          <input
            className="field__input"
            value={second}
            onChange={(e) => setSecond(e.target.value)}
            placeholder={secondPlaceholder}
          />
        )}
        <button className="btn btn--primary btn--small" onClick={handleClick}>{label}</button>
      </div>
      {msg && <div className={`feedback ${msg.ok ? 'feedback--ok' : 'feedback--error'}`}>{msg.text}</div>}
    </div>
  )
}

export default function FileSystemPanel({ onActivity }) {
  const [mkdirPath, setMkdirPath] = useState('')
  const [mkdirP, setMkdirP] = useState(true)
  const [mkdirMsg, setMkdirMsg] = useState(null)

  const [mkfilePath, setMkfilePath] = useState('')
  const [mkfileSize, setMkfileSize] = useState('')
  const [mkfileCont, setMkfileCont] = useState('')
  const [mkfileR, setMkfileR] = useState(false)
  const [mkfileMsg, setMkfileMsg] = useState(null)

  const [editPath, setEditPath] = useState('')
  const [editContenido, setEditContenido] = useState('')
  const [editMsg, setEditMsg] = useState(null)

  const [copySrc, setCopySrc] = useState('')
  const [copyDst, setCopyDst] = useState('')
  const [copyMsg, setCopyMsg] = useState(null)

  const [moveSrc, setMoveSrc] = useState('')
  const [moveDst, setMoveDst] = useState('')
  const [moveMsg, setMoveMsg] = useState(null)

  async function exec(line) {
    const { output } = await runCommand(line)
    onActivity?.()
    return { output }
  }

  async function handleMkdir(e) {
    e.preventDefault()
    try {
      const { output } = await exec(buildLine('mkdir', { path: mkdirPath, p: mkdirP }))
      setMkdirMsg({ ok: true, text: output })
    } catch (err) {
      setMkdirMsg({ ok: false, text: err.message })
    }
  }

  async function handleMkfile(e) {
    e.preventDefault()
    try {
      const { output } = await exec(
        buildLine('mkfile', { path: mkfilePath, size: mkfileSize, cont: mkfileCont, r: mkfileR })
      )
      setMkfileMsg({ ok: true, text: output })
    } catch (err) {
      setMkfileMsg({ ok: false, text: err.message })
    }
  }

  async function handleEdit(e) {
    e.preventDefault()
    try {
      const { output } = await exec(buildLine('edit', { path: editPath, contenido: editContenido }))
      setEditMsg({ ok: true, text: output })
    } catch (err) {
      setEditMsg({ ok: false, text: err.message })
    }
  }

  async function handleCopy(e) {
    e.preventDefault()
    try {
      const { output } = await exec(buildLine('copy', { path: copySrc, destino: copyDst }))
      setCopyMsg({ ok: true, text: output })
    } catch (err) {
      setCopyMsg({ ok: false, text: err.message })
    }
  }

  async function handleMove(e) {
    e.preventDefault()
    try {
      const { output } = await exec(buildLine('move', { path: moveSrc, destino: moveDst }))
      setMoveMsg({ ok: true, text: output })
    } catch (err) {
      setMoveMsg({ ok: false, text: err.message })
    }
  }

  return (
    <div className="fs-panel">
      <div className="card">
        <h3 className="card__title">Crear carpeta (MKDIR)</h3>
        <form onSubmit={handleMkdir}>
          <div className="form-grid">
            <div className="field">
              <label className="field__label">Ruta</label>
              <input className="field__input" value={mkdirPath} onChange={(e) => setMkdirPath(e.target.value)} placeholder="/home/usr/docs" />
            </div>
          </div>
          <label className="fs-panel__checkbox">
            <input type="checkbox" checked={mkdirP} onChange={(e) => setMkdirP(e.target.checked)} />
            Crear carpetas padre faltantes (-p)
          </label>
          <button className="btn btn--primary" type="submit" style={{ marginTop: 10 }}>Crear</button>
        </form>
        {mkdirMsg && <div className={`feedback ${mkdirMsg.ok ? 'feedback--ok' : 'feedback--error'}`}>{mkdirMsg.text}</div>}
      </div>

      <div className="card">
        <h3 className="card__title">Crear archivo (MKFILE)</h3>
        <form onSubmit={handleMkfile}>
          <div className="form-grid">
            <div className="field">
              <label className="field__label">Ruta</label>
              <input className="field__input" value={mkfilePath} onChange={(e) => setMkfilePath(e.target.value)} placeholder="/home/usr/docs/nota.txt" />
            </div>
            <div className="field">
              <label className="field__label">Tamaño (bytes)</label>
              <input className="field__input" value={mkfileSize} onChange={(e) => setMkfileSize(e.target.value)} placeholder="0" />
            </div>
            <div className="field">
              <label className="field__label">Contenido literal</label>
              <input className="field__input" value={mkfileCont} onChange={(e) => setMkfileCont(e.target.value)} placeholder="opcional" />
            </div>
          </div>
          <label className="fs-panel__checkbox">
            <input type="checkbox" checked={mkfileR} onChange={(e) => setMkfileR(e.target.checked)} />
            Crear carpetas padre faltantes (-r)
          </label>
          <button className="btn btn--primary" type="submit" style={{ marginTop: 10 }}>Crear</button>
        </form>
        {mkfileMsg && <div className={`feedback ${mkfileMsg.ok ? 'feedback--ok' : 'feedback--error'}`}>{mkfileMsg.text}</div>}
      </div>

      <div className="card">
        <h3 className="card__title">Eliminar (RM / REMOVE)</h3>
        <p className="fs-panel__hint">RM solo borra carpetas vacías. REMOVE borra recursivamente todo el contenido.</p>
        <ActionRow label="RM" placeholder="/home/usr/docs/vacia" onRun={(v) => exec(buildLine('rm', { path: v }))} />
        <ActionRow label="REMOVE (recursivo)" placeholder="/home/usr/docs" onRun={(v) => exec(buildLine('remove', { path: v }))} />
      </div>

      <div className="card">
        <h3 className="card__title">Ver contenido (CAT)</h3>
        <ActionRow label="CAT" placeholder="/home/usr/docs/nota.txt" onRun={(v) => exec(buildLine('cat', { file1: v }))} />
      </div>

      <div className="card">
        <h3 className="card__title">Renombrar (RENAME)</h3>
        <ActionRow
          label="Renombrar"
          placeholder="/home/usr/docs/nota.txt (ruta actual)"
          secondPlaceholder="nuevo_nombre.txt"
          twoFields
          onRun={(v, second) => exec(buildLine('rename', { path: v, name: second }))}
        />
      </div>

      <div className="card">
        <h3 className="card__title">Editar contenido (EDIT)</h3>
        <p className="fs-panel__hint">Reemplaza el contenido del archivo con el de un archivo existente en el servidor.</p>
        <form onSubmit={handleEdit}>
          <div className="form-grid">
            <div className="field">
              <label className="field__label">Archivo en EXT2</label>
              <input className="field__input" value={editPath} onChange={(e) => setEditPath(e.target.value)} placeholder="/home/usr/docs/nota.txt" />
            </div>
            <div className="field">
              <label className="field__label">Archivo fuente (en el servidor)</label>
              <input className="field__input" value={editContenido} onChange={(e) => setEditContenido(e.target.value)} placeholder="/home/usr/nuevo_contenido.txt" />
            </div>
          </div>
          <button className="btn btn--primary" type="submit">Editar</button>
        </form>
        {editMsg && <div className={`feedback ${editMsg.ok ? 'feedback--ok' : 'feedback--error'}`}>{editMsg.text}</div>}
      </div>

      <div className="card">
        <h3 className="card__title">Copiar (COPY)</h3>
        <form onSubmit={handleCopy}>
          <div className="form-grid">
            <div className="field">
              <label className="field__label">Origen</label>
              <input className="field__input" value={copySrc} onChange={(e) => setCopySrc(e.target.value)} placeholder="/home/usr/docs" />
            </div>
            <div className="field">
              <label className="field__label">Carpeta destino</label>
              <input className="field__input" value={copyDst} onChange={(e) => setCopyDst(e.target.value)} placeholder="/home/usr/backup" />
            </div>
          </div>
          <button className="btn btn--primary" type="submit">Copiar</button>
        </form>
        {copyMsg && <div className={`feedback ${copyMsg.ok ? 'feedback--ok' : 'feedback--error'}`}>{copyMsg.text}</div>}
      </div>

      <div className="card">
        <h3 className="card__title">Mover (MOVE)</h3>
        <form onSubmit={handleMove}>
          <div className="form-grid">
            <div className="field">
              <label className="field__label">Origen</label>
              <input className="field__input" value={moveSrc} onChange={(e) => setMoveSrc(e.target.value)} placeholder="/home/usr/docs/nota.txt" />
            </div>
            <div className="field">
              <label className="field__label">Carpeta destino</label>
              <input className="field__input" value={moveDst} onChange={(e) => setMoveDst(e.target.value)} placeholder="/home/usr/archivados" />
            </div>
          </div>
          <button className="btn btn--primary" type="submit">Mover</button>
        </form>
        {moveMsg && <div className={`feedback ${moveMsg.ok ? 'feedback--ok' : 'feedback--error'}`}>{moveMsg.text}</div>}
      </div>

      <style>{`
        .fs-panel {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(360px, 1fr));
          gap: 16px;
          align-items: start;
        }
        .fs-panel__checkbox {
          display: flex;
          align-items: center;
          gap: 7px;
          font-size: 12.5px;
          color: var(--text-dim);
        }
        .fs-panel__hint {
          font-size: 12px;
          color: var(--text-faint);
          margin: -6px 0 12px 0;
        }
        .action-row { margin-bottom: 14px; }
        .action-row:last-child { margin-bottom: 0; }
        .action-row__fields {
          display: flex;
          gap: 8px;
        }
        .action-row__fields .field__input { flex: 1; }
      `}</style>
    </div>
  )
}
