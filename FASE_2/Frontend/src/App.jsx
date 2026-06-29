
import React, { useCallback, useEffect, useState } from 'react'
import StatusBar from './components/StatusBar.jsx'
import DiskPanel from './components/DiskPanel.jsx'
import LoginPanel from './components/LoginPanel.jsx'
import FileSystemPanel from './components/FileSystemPanel.jsx'
import UsersPanel from './components/UsersPanel.jsx'
import ReportsPanel from './components/ReportsPanel.jsx'
import Console from './components/Console.jsx'
import { getSession } from './api/client.js'

const TABS = [
  { id: 'discos', label: 'Discos y particiones' },
  { id: 'sesion', label: 'Sesión' },
  { id: 'archivos', label: 'Archivos y carpetas' },
  { id: 'usuarios', label: 'Grupos y usuarios' },
  { id: 'reportes', label: 'Reportes' },
  { id: 'consola', label: 'Consola' },
]

export default function App() {
  const [tab, setTab] = useState('discos')
  const [session, setSession] = useState(null)
  const [backendOnline, setBackendOnline] = useState(true)

  const refreshSession = useCallback(async () => {
    try {
      const s = await getSession()
      setSession(s)
      setBackendOnline(true)
    } catch {
      setBackendOnline(false)
    }
  }, [])

  useEffect(() => {
    refreshSession()
    const interval = setInterval(refreshSession, 5000)
    return () => clearInterval(interval)
  }, [refreshSession])

  return (
    <div className="app">
      <StatusBar session={session} backendOnline={backendOnline} />

      <nav className="app__tabs">
        {TABS.map((t) => (
          <button
            key={t.id}
            className={`app__tab ${tab === t.id ? 'app__tab--active' : ''}`}
            onClick={() => setTab(t.id)}
          >
            {t.label}
          </button>
        ))}
      </nav>

      <main className="app__content">
        {tab === 'discos' && <DiskPanel onActivity={refreshSession} />}
        {tab === 'sesion' && <LoginPanel session={session} onActivity={refreshSession} />}
        {tab === 'archivos' && <FileSystemPanel onActivity={refreshSession} />}
        {tab === 'usuarios' && <UsersPanel onActivity={refreshSession} />}
        {tab === 'reportes' && <ReportsPanel session={session} />}
        {tab === 'consola' && (
          <div className="app__console-wrap">
            <Console onActivity={refreshSession} />
          </div>
        )}
      </main>

      <style>{`
        .app {
          min-height: 100vh;
          display: flex;
          flex-direction: column;
        }
        .app__tabs {
          display: flex;
          gap: 4px;
          padding: 14px 24px 0;
          overflow-x: auto;
        }
        .app__tab {
          background: transparent;
          border: 1px solid var(--line);
          border-bottom: none;
          color: var(--text-dim);
          padding: 9px 16px;
          font-size: 12.5px;
          font-weight: 500;
          border-radius: var(--radius) var(--radius) 0 0;
          cursor: pointer;
          white-space: nowrap;
        }
        .app__tab--active {
          color: var(--phosphor);
          background: var(--bg-panel);
          border-color: var(--line-bright);
        }
        .app__content {
          flex: 1;
          padding: 24px;
          background: var(--bg-panel);
          border-top: 1px solid var(--line);
        }
        .app__console-wrap {
          height: calc(100vh - 200px);
          min-height: 420px;
        }
      `}</style>
    </div>
  )
}
