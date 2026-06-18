import { reactive, onMounted, onUnmounted } from 'vue'
import { getAuth, getCredentials, apiUrl, wsUrl } from '../utils.js'
import { normalizeHistoryPoint } from './useChartProcessing.js'

// Intentional singleton: all useWebSocket() calls share one connection.
// This works for the current single-page app with a single mount point.
// If the app grows to multiple mount points, migrate to provide/inject.

const state = reactive({
  agents: new Map(),
  connected: false,
  // Wall clock, ticked once a second while the realtime connection is mounted.
  // Components derive each agent's freshness from this against last_seen, so a
  // silent agent (no more WS ticks) still flips to offline on its own.
  now: Date.now(),
})

// Mirror of the monitor's offlineThreshold (internal/monitor/store.go). An agent
// is online while its last report is newer than this.
export const OFFLINE_THRESHOLD_MS = 30000

// isAgentOnline derives liveness on the client from last_seen vs the ticking
// clock, instead of trusting the snapshot's `online` flag — that flag is frozen
// at the value of the last received tick and never updates once reports stop.
// last_seen is RFC3339 with an offset, so Date.parse yields a correct absolute
// instant regardless of the monitor's or browser's timezone.
export function isAgentOnline(agent, now) {
  if (!agent?.last_seen) return false
  return now - Date.parse(agent.last_seen) < OFFLINE_THRESHOLD_MS
}

let clockTimer = null

const historyListeners = new Map()
const historyInflight = new Map()

let ws = null
let reconnectTimer = null

// Forward each live point to the charts subscribed for this agent. Charts own
// their own series (seeded from getHistory), so nothing is buffered here.
function appendHistoryPoint(snap, historyPoint) {
  const listeners = historyListeners.get(snap.agent_id)
  if (!listeners) return
  const pt = normalizeHistoryPoint(snap, historyPoint)
  for (const fn of listeners) fn(pt)
}

async function fetchHistory(agentId) {
  const resp = await fetch(apiUrl(`/api/v1/agents/${encodeURIComponent(agentId)}/history`), {
    headers: { 'Authorization': `Basic ${getAuth()}` },
  })
  if (!resp.ok) return []
  const data = await resp.json()
  return (data.points || []).map(p => ({ ...p, t: p.t * 1000 }))
}

function connect() {
  const { user, pass } = getCredentials()
  const url = wsUrl('/ws')
  url.search = `?user=${encodeURIComponent(user)}&pass=${encodeURIComponent(pass)}`

  ws = new WebSocket(url)

  ws.onopen = () => {
    state.connected = true
    console.log('WebSocket connected')
  }

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      if (msg.type === 'snapshot' && msg.data) {
        state.agents.set(msg.data.agent_id, msg.data)
        appendHistoryPoint(msg.data, msg.history)
      }
    } catch (e) {
      console.error('WS parse error:', e)
    }
  }

  ws.onclose = () => {
    state.connected = false
    console.log('WebSocket disconnected, reconnecting in 3s...')
    reconnectTimer = setTimeout(connect, 3000)
  }

  ws.onerror = (err) => {
    console.error('WebSocket error:', err)
    ws.close()
  }
}

function fetchAgents() {
  fetch(apiUrl('/api/v1/agents'), {
    headers: { 'Authorization': `Basic ${getAuth()}` },
  })
    .then(r => r.json())
    .then(agents => {
      if (Array.isArray(agents)) {
        agents.forEach(s => state.agents.set(s.agent_id, s))
      }
    })
    .catch(e => console.error('Fetch agents error:', e))
}

// getHistory loads an agent's entire stored history (the agent returns the
// whole buffer when no window is given) and the charts slice it client-side.
// Concurrent callers — the several charts of one agent mounting together —
// share a single in-flight request; each gets its own array copy so their
// independent live appends don't collide.
async function getHistory(agentId) {
  if (!historyInflight.has(agentId)) {
    const p = fetchHistory(agentId).finally(() => historyInflight.delete(agentId))
    historyInflight.set(agentId, p)
  }
  const points = await historyInflight.get(agentId)
  return points.slice()
}

function onHistoryUpdate(agentId, callback) {
  if (!historyListeners.has(agentId)) {
    historyListeners.set(agentId, new Set())
  }
  historyListeners.get(agentId).add(callback)
  return () => historyListeners.get(agentId)?.delete(callback)
}

// useRealtimeConnection owns the singleton connection lifecycle. Mount it once,
// at the app root — it opens the socket on mount and stops reconnect attempts
// on teardown.
export function useRealtimeConnection() {
  onMounted(() => {
    if (!ws || ws.readyState === WebSocket.CLOSED) {
      connect()
    }
    if (!clockTimer) {
      clockTimer = setInterval(() => { state.now = Date.now() }, 1000)
    }
  })

  onUnmounted(() => {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
    }
    if (clockTimer) {
      clearInterval(clockTimer)
      clockTimer = null
    }
  })
}

// useWebSocket is a side-effect-free accessor to the shared realtime state and
// helpers. Safe to call from any component, any number of times — it registers
// no lifecycle hooks, so consumers reach the singleton directly rather than
// having these threaded down as props.
export function useWebSocket() {
  return { state, fetchAgents, getHistory, onHistoryUpdate }
}
