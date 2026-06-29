// Archivo: src/components/Console.jsx
import React, { useEffect, useRef, useState } from 'react'
import { runCommand, runScript, uploadScript } from '../api/client.js'

// Console: el corazon de la app. Reproduce la consola interactiva de la
// Fase 1 (un comando, una salida) pero en el navegador, y ademas permite
// pegar o subir un script .smia completo, igual que el comando EXEC.
export default function Console({ onActivity }) {
  const [history, setHistory] = useState([
    { type: 'system', text: 'Consola lista. Escriba un comando MIA y presione Enter.' },
  ])
  const [input, setInput] = useState('')
  const [busy, setBusy] = useState(false)
  const [mode, setMode] = useState('line') // 'line' | 'script'
  const [scriptText, setScriptText] = useState('')
  const scrollRef = useRef(null)
  const fileInputRef = useRef(null)

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [history])

  async function executeLine(line) {
    if (!line.trim()) return
    setHistory((h) => [...h, { type: 'input', text: line }])
    setBusy(true)
    try {
      const { output } = await runCommand(line)
      setHistory((h) => [...h, { type: 'output', text: output || '(sin salida)' }])
      onActivity?.()
    } catch (err) {
      setHistory((h) => [...h, { type: 'error', text: err.message }])
    } finally {
      setBusy(false)
    }
  }

  async function executeScript() {
    if (!scriptText.trim()) return
    setHistory((h) => [...h, { type: 'input', text: `[script pegado, ${scriptText.split('\n').length} lineas]` }])
    setBusy(true)
    try {
      const { output } = await runScript(scriptText)
      setHistory((h) => [...h, { type: 'output', text: output || '(sin salida)' }])
      onActivity?.()
    } catch (err) {
      setHistory((h) => [...h, { type: 'error', text: err.message }])
    } finally {
      setBusy(false)
    }
  }

  async function handleFileUpload(e) {
    const file = e.target.files?.[0]
    if (!file) return
    setHistory((h) => [...h, { type: 'input', text: `[archivo subido: ${file.name}]` }])
    setBusy(true)
    try {
      const { output } = await uploadScript(file)
      setHistory((h) => [...h, { type: 'output', text: output || '(sin salida)' }])
      onActivity?.()
    } catch (err) {
      setHistory((h) => [...h, { type: 'error', text: err.message }])
    } finally {
      setBusy(false)
      e.target.value = ''
    }
  }

  function handleKeyDown(e) {
    if (e.key === 'Enter' && !busy) {
      const line = input
      setInput('')
      executeLine(line)
    }
  }

  return (
    <section className="console">
      <div className="console__tabs">
        <button
          className={`console__tab ${mode === 'line' ? 'console__tab--active' : ''}`}
          onClick={() => setMode('line')}
        >
          comando unico
        </button>
        <button
          className={`console__tab ${mode === 'script' ? 'console__tab--active' : ''}`}
          onClick={() => setMode('script')}
        >
          script .smia
        </button>
      </div>

      <div className="console__screen" ref={scrollRef}>
        {history.map((entry, i) => (
          <div key={i} className={`console__line console__line--${entry.type}`}>
            {entry.type === 'input' && <span className="console__prompt">mia&gt;</span>}
            <pre className="console__text">{entry.text}</pre>
          </div>
        ))}
        {busy && <div className="console__line console__line--system">procesando…</div>}
      </div>

      {mode === 'line' ? (
        <div className="console__input-row">
          <span className="console__prompt">mia&gt;</span>
          <input
            className="console__input"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder='mkdisk -size=10 -unit=M -path=/home/usr/disco1.mia'
            disabled={busy}
            autoFocus
            spellCheck={false}
          />
        </div>
      ) : (
        <div className="console__script-row">
          <textarea
            className="console__textarea"
            value={scriptText}
            onChange={(e) => setScriptText(e.target.value)}
            placeholder={'# Pegue aqui un script .smia completo\nmkdisk -size=10 -unit=M -path=/home/usr/disco1.mia\nfdisk -size=5 -unit=M -path=/home/usr/disco1.mia -name=Part1'}
            spellCheck={false}
            rows={6}
          />
          <div className="console__script-actions">
            <button className="btn btn--primary" disabled={busy} onClick={executeScript}>
              Ejecutar script
            </button>
            <button
              className="btn btn--ghost"
              disabled={busy}
              onClick={() => fileInputRef.current?.click()}
            >
              Subir archivo .smia
            </button>
            <input
              ref={fileInputRef}
              type="file"
              accept=".smia,.txt"
              style={{ display: 'none' }}
              onChange={handleFileUpload}
            />
          </div>
        </div>
      )}

      <style>{`
        .console {
          display: flex;
          flex-direction: column;
          background: var(--bg-panel);
          border: 1px solid var(--line);
          border-radius: var(--radius);
          overflow: hidden;
          height: 100%;
        }
        .console__tabs {
          display: flex;
          border-bottom: 1px solid var(--line);
          background: var(--bg-raised);
        }
        .console__tab {
          background: transparent;
          border: none;
          color: var(--text-dim);
          font-size: 12px;
          font-family: var(--font-mono);
          letter-spacing: 0.03em;
          padding: 10px 16px;
          cursor: pointer;
          border-bottom: 2px solid transparent;
        }
        .console__tab--active {
          color: var(--phosphor);
          border-bottom-color: var(--phosphor);
        }
        .console__screen {
          flex: 1;
          overflow-y: auto;
          padding: 16px 18px;
          font-family: var(--font-mono);
          font-size: 13px;
          line-height: 1.55;
          min-height: 260px;
        }
        .console__line {
          margin-bottom: 6px;
          display: flex;
          gap: 8px;
          align-items: flex-start;
        }
        .console__text {
          margin: 0;
          white-space: pre-wrap;
          word-break: break-word;
          font-family: var(--font-mono);
        }
        .console__line--input .console__text { color: var(--text); }
        .console__line--output .console__text { color: var(--text-dim); }
        .console__line--error .console__text { color: var(--danger); }
        .console__line--system .console__text { color: var(--text-faint); font-style: italic; }
        .console__prompt {
          color: var(--phosphor);
          flex-shrink: 0;
          font-weight: 700;
        }
        .console__input-row {
          display: flex;
          align-items: center;
          gap: 8px;
          padding: 12px 18px;
          border-top: 1px solid var(--line);
          background: var(--bg-raised);
        }
        .console__input {
          flex: 1;
          background: transparent;
          border: none;
          color: var(--text);
          font-family: var(--font-mono);
          font-size: 13px;
          outline: none;
        }
        .console__input::placeholder { color: var(--text-faint); }
        .console__script-row {
          padding: 14px 18px;
          border-top: 1px solid var(--line);
          background: var(--bg-raised);
          display: flex;
          flex-direction: column;
          gap: 10px;
        }
        .console__textarea {
          background: var(--bg);
          border: 1px solid var(--line);
          border-radius: var(--radius);
          color: var(--text);
          font-family: var(--font-mono);
          font-size: 12.5px;
          padding: 10px 12px;
          resize: vertical;
          outline: none;
        }
        .console__textarea:focus { border-color: var(--phosphor-dim); }
        .console__script-actions {
          display: flex;
          gap: 10px;
        }
      `}</style>
    </section>
  )
}
