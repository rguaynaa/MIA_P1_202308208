// Archivo: src/components/StatusBar.jsx
import React from 'react'

// StatusBar: franja superior persistente que muestra el estado de conexion
// con el backend y la sesion activa (LOGIN). Funciona como el "indicador de
// encendido" del panel: siempre visible, nunca decorativo.
export default function StatusBar({ session, backendOnline }) {
  return (
    <header className="status-bar">
      <div className="status-bar__brand">
        <span className="status-bar__dot" aria-hidden="true" />
        <span className="status-bar__title">MIA</span>
        <span className="status-bar__subtitle">control EXT2 · fase 2</span>
      </div>

      <div className="status-bar__right">
        <div className={`status-pill ${backendOnline ? 'status-pill--ok' : 'status-pill--off'}`}>
          <span className="status-pill__dot" />
          {backendOnline ? 'backend conectado' : 'backend sin respuesta'}
        </div>

        {session?.loggedIn ? (
          <div className="status-pill status-pill--session">
            <span className="mono">{session.user}</span>
            <span className="status-bar__sep">/</span>
            <span className="mono">{session.id}</span>
            {session.isRoot && <span className="status-bar__root-tag">root</span>}
          </div>
        ) : (
          <div className="status-pill status-pill--idle">sin sesion</div>
        )}
      </div>

      <style>{`
        .status-bar {
          display: flex;
          align-items: center;
          justify-content: space-between;
          gap: 16px;
          padding: 14px 24px;
          background: var(--bg-panel);
          border-bottom: 1px solid var(--line);
          position: sticky;
          top: 0;
          z-index: 50;
        }
        .status-bar__brand {
          display: flex;
          align-items: baseline;
          gap: 10px;
        }
        .status-bar__dot {
          width: 9px;
          height: 9px;
          border-radius: 50%;
          background: var(--phosphor);
          box-shadow: 0 0 8px 1px var(--phosphor-glow);
          display: inline-block;
        }
        .status-bar__title {
          font-weight: 700;
          font-size: 18px;
          letter-spacing: 0.04em;
          color: var(--text);
        }
        .status-bar__subtitle {
          font-size: 12px;
          color: var(--text-dim);
          letter-spacing: 0.03em;
        }
        .status-bar__right {
          display: flex;
          align-items: center;
          gap: 10px;
        }
        .status-pill {
          display: flex;
          align-items: center;
          gap: 6px;
          padding: 5px 12px;
          border-radius: 999px;
          font-size: 12px;
          font-family: var(--font-mono);
          border: 1px solid var(--line);
          color: var(--text-dim);
        }
        .status-pill__dot {
          width: 6px;
          height: 6px;
          border-radius: 50%;
        }
        .status-pill--ok { color: var(--phosphor); border-color: var(--phosphor-dim); }
        .status-pill--ok .status-pill__dot { background: var(--phosphor); }
        .status-pill--off { color: var(--danger); border-color: var(--danger-dim); }
        .status-pill--off .status-pill__dot { background: var(--danger); }
        .status-pill--session { color: var(--text); border-color: var(--line-bright); }
        .status-pill--idle { color: var(--text-faint); }
        .status-bar__sep { color: var(--text-faint); }
        .status-bar__root-tag {
          background: var(--amber-dim);
          color: var(--amber);
          padding: 1px 6px;
          border-radius: 3px;
          font-size: 10px;
          text-transform: uppercase;
          letter-spacing: 0.05em;
        }
      `}</style>
    </header>
  )
}
