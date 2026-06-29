// Archivo: src/components/LoginPanel.jsx
import React, { useEffect, useState } from 'react'
import { runCommand, getMounted } from '../api/client.js'
import './shared.css'

export default function LoginPanel({ session, onActivity }) {
  const [user, setUser] = useState('root')
  const [pass, setPass] = useState('123')
  const [id, setId] = useState('')
  const [msg, setMsg] = useState(null)
  const [mounted, setMounted] = useState([])
  const [loadingMounted, setLoadingMounted] = useState(false)

  async function loadMounted() {
    setLoadingMounted(true)
    try {
      const list = await getMounted()
      setMounted(list)
    } catch {
      // backend caido; se ignora, el StatusBar ya lo refleja
    } finally {
      setLoadingMounted(false)
    }
  }

  useEffect(() => {
    loadMounted()
  }, [session])

  async function handleLogin(e) {
    e.preventDefault()
    const line = `login -user=${user} -pass=${pass} -id=${id}`
    try {
      const { output } = await runCommand(line)
      const isError = output.toLowerCase().includes('error')
      if (isError && output.toLowerCase().includes('users.txt')) {
        // El backend devuelve "Error: no se pudo leer users.txt" cuando la
        // particion esta montada pero nunca se formateo (falta MKFS), que
        // es el unico archivo que crea el usuario root inicial.
        setMsg({
          ok: false,
          text: 'Esta partición no ha sido formateada todavía. Ve a "Discos y particiones" y ejecuta MKFS sobre este ID antes de iniciar sesión.',
        })
      } else {
        setMsg({ ok: !isError, text: output })
      }
      onActivity?.()
    } catch (err) {
      setMsg({ ok: false, text: err.message })
    }
  }

  async function handleLogout() {
    try {
      const { output } = await runCommand('logout')
      setMsg({ ok: true, text: output })
      onActivity?.()
    } catch (err) {
      setMsg({ ok: false, text: err.message })
    }
  }

  return (
    <div className="login-panel">
      <div className="card">
        <h3 className="card__title">Sesión (LOGIN / LOGOUT)</h3>

        {session?.loggedIn ? (
          <div>
            <p className="login-panel__active">
              Sesión activa como <strong className="mono">{session.user}</strong> en{' '}
              <span className="mono">{session.id}</span>
              {session.isRoot && <span className="tag tag--primary" style={{ marginLeft: 8 }}>root</span>}
            </p>
            <button className="btn btn--danger" onClick={handleLogout}>Cerrar sesión</button>
          </div>
        ) : (
          <form onSubmit={handleLogin}>
            <div className="form-grid">
              <div className="field">
                <label className="field__label">Usuario</label>
                <input className="field__input" value={user} onChange={(e) => setUser(e.target.value)} />
              </div>
              <div className="field">
                <label className="field__label">Contraseña</label>
                <input className="field__input" type="password" value={pass} onChange={(e) => setPass(e.target.value)} />
              </div>
              <div className="field">
                <label className="field__label">ID de partición</label>
                <select className="field__select" value={id} onChange={(e) => setId(e.target.value)}>
                  <option value="">Seleccione…</option>
                  {mounted.map((mp) => (
                    <option key={mp.id} value={mp.id}>
                      {mp.id} · {mp.name}
                    </option>
                  ))}
                </select>
              </div>
            </div>
            <button className="btn btn--primary" type="submit">Iniciar sesión</button>
          </form>
        )}

        {msg && <div className={`feedback ${msg.ok ? 'feedback--ok' : 'feedback--error'}`}>{msg.text}</div>}
      </div>

      <div className="card">
        <h3 className="card__title">Particiones montadas (MOUNTED)</h3>
        <button className="btn btn--small" onClick={loadMounted} disabled={loadingMounted}>
          {loadingMounted ? 'actualizando…' : 'actualizar lista'}
        </button>
        <table className="table" style={{ marginTop: 14 }}>
          <thead>
            <tr>
              <th>ID</th>
              <th>Disco</th>
              <th>Partición</th>
              <th>Correlativo</th>
            </tr>
          </thead>
          <tbody>
            {mounted.length === 0 && (
              <tr><td colSpan={4} style={{ color: 'var(--text-faint)' }}>No hay particiones montadas</td></tr>
            )}
            {mounted.map((mp) => (
              <tr key={mp.id}>
                <td className="mono">{mp.id}</td>
                <td className="mono">{mp.path}</td>
                <td className="mono">{mp.name}</td>
                <td>{mp.correlative}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <style>{`
        .login-panel {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(360px, 1fr));
          gap: 16px;
        }
        .login-panel__active {
          color: var(--text-dim);
          font-size: 13px;
          margin-bottom: 12px;
        }
      `}</style>
    </div>
  )
}
