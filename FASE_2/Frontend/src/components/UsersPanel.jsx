// Archivo: src/components/UsersPanel.jsx
import React, { useState } from 'react'
import { runCommand } from '../api/client.js'
import './shared.css'

function buildLine(cmd, flags) {
  const parts = [cmd]
  for (const [key, val] of Object.entries(flags)) {
    if (val === undefined || val === null || val === '') continue
    parts.push(`-${key}=${val}`)
  }
  return parts.join(' ')
}

export default function UsersPanel({ onActivity }) {
  const [grpName, setGrpName] = useState('')
  const [grpMsg, setGrpMsg] = useState(null)

  const [rmGrpName, setRmGrpName] = useState('')
  const [rmGrpMsg, setRmGrpMsg] = useState(null)

  const [usrUser, setUsrUser] = useState('')
  const [usrPass, setUsrPass] = useState('')
  const [usrGrp, setUsrGrp] = useState('')
  const [usrMsg, setUsrMsg] = useState(null)

  const [rmUsrUser, setRmUsrUser] = useState('')
  const [rmUsrMsg, setRmUsrMsg] = useState(null)

  const [chgrpUser, setChgrpUser] = useState('')
  const [chgrpGrp, setChgrpGrp] = useState('')
  const [chgrpMsg, setChgrpMsg] = useState(null)

  async function run(line, setMsg) {
    try {
      const { output } = await runCommand(line)
      setMsg({ ok: !output.toLowerCase().includes('error'), text: output })
      onActivity?.()
    } catch (err) {
      setMsg({ ok: false, text: err.message })
    }
  }

  return (
    <div className="users-panel">
      <div className="card">
        <h3 className="card__title">Grupos (MKGRP / RMGRP)</h3>
        <div className="users-panel__row">
          <input className="field__input" value={grpName} onChange={(e) => setGrpName(e.target.value)} placeholder="nombre del grupo" />
          <button className="btn btn--primary btn--small" onClick={() => run(buildLine('mkgrp', { name: grpName }), setGrpMsg)}>
            Crear
          </button>
        </div>
        {grpMsg && <div className={`feedback ${grpMsg.ok ? 'feedback--ok' : 'feedback--error'}`}>{grpMsg.text}</div>}

        <div className="users-panel__row" style={{ marginTop: 14 }}>
          <input className="field__input" value={rmGrpName} onChange={(e) => setRmGrpName(e.target.value)} placeholder="nombre del grupo a eliminar" />
          <button className="btn btn--danger btn--small" onClick={() => run(buildLine('rmgrp', { name: rmGrpName }), setRmGrpMsg)}>
            Eliminar
          </button>
        </div>
        {rmGrpMsg && <div className={`feedback ${rmGrpMsg.ok ? 'feedback--ok' : 'feedback--error'}`}>{rmGrpMsg.text}</div>}
      </div>

      <div className="card">
        <h3 className="card__title">Crear usuario (MKUSR)</h3>
        <div className="form-grid">
          <div className="field">
            <label className="field__label">Usuario</label>
            <input className="field__input" value={usrUser} onChange={(e) => setUsrUser(e.target.value)} />
          </div>
          <div className="field">
            <label className="field__label">Contraseña</label>
            <input className="field__input" value={usrPass} onChange={(e) => setUsrPass(e.target.value)} />
          </div>
          <div className="field">
            <label className="field__label">Grupo</label>
            <input className="field__input" value={usrGrp} onChange={(e) => setUsrGrp(e.target.value)} />
          </div>
        </div>
        <button
          className="btn btn--primary"
          onClick={() => run(buildLine('mkusr', { user: usrUser, pass: usrPass, grp: usrGrp }), setUsrMsg)}
        >
          Crear usuario
        </button>
        {usrMsg && <div className={`feedback ${usrMsg.ok ? 'feedback--ok' : 'feedback--error'}`}>{usrMsg.text}</div>}
      </div>

      <div className="card">
        <h3 className="card__title">Eliminar usuario (RMUSR)</h3>
        <div className="users-panel__row">
          <input className="field__input" value={rmUsrUser} onChange={(e) => setRmUsrUser(e.target.value)} placeholder="nombre de usuario" />
          <button className="btn btn--danger btn--small" onClick={() => run(buildLine('rmusr', { user: rmUsrUser }), setRmUsrMsg)}>
            Eliminar
          </button>
        </div>
        {rmUsrMsg && <div className={`feedback ${rmUsrMsg.ok ? 'feedback--ok' : 'feedback--error'}`}>{rmUsrMsg.text}</div>}
      </div>

      <div className="card">
        <h3 className="card__title">Cambiar grupo de usuario (CHGRP)</h3>
        <div className="form-grid">
          <div className="field">
            <label className="field__label">Usuario</label>
            <input className="field__input" value={chgrpUser} onChange={(e) => setChgrpUser(e.target.value)} />
          </div>
          <div className="field">
            <label className="field__label">Nuevo grupo</label>
            <input className="field__input" value={chgrpGrp} onChange={(e) => setChgrpGrp(e.target.value)} />
          </div>
        </div>
        <button
          className="btn btn--primary"
          onClick={() => run(buildLine('chgrp', { user: chgrpUser, grp: chgrpGrp }), setChgrpMsg)}
        >
          Cambiar grupo
        </button>
        {chgrpMsg && <div className={`feedback ${chgrpMsg.ok ? 'feedback--ok' : 'feedback--error'}`}>{chgrpMsg.text}</div>}
      </div>

      <style>{`
        .users-panel {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(360px, 1fr));
          gap: 16px;
          align-items: start;
        }
        .users-panel__row {
          display: flex;
          gap: 8px;
        }
        .users-panel__row .field__input { flex: 1; }
      `}</style>
    </div>
  )
}
