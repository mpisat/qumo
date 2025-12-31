import { useConfig } from './config.tsx'
import './App.css'

function App() {
  const { config, setConfig } = useConfig()

  return (
    <div class="app">
      <h1>{config.appName}</h1>
      <div class="card">
        <p>Relay URL: <code>{config.relayUrl}</code></p>
        <p>API URL: <code>{config.apiUrl}</code></p>
        <p>Mode: <code>{config.isDev ? 'Development' : 'Production'}</code></p>
      </div>
      <div class="card">
        <label>
          Relay URL:
          <input
            type="text"
            value={config.relayUrl}
            onInput={(e) => setConfig("relayUrl", e.currentTarget.value)}
          />
        </label>
      </div>
    </div>
  )
}

export default App
