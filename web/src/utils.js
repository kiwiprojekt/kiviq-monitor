export function getCredentials() {
  return {
    user: sessionStorage.getItem('kiviq_user') || '',
    pass: sessionStorage.getItem('kiviq_pass') || '',
  }
}

export function setCredentials(user, pass) {
  sessionStorage.setItem('kiviq_user', user)
  sessionStorage.setItem('kiviq_pass', pass)
}

export function clearCredentials() {
  sessionStorage.removeItem('kiviq_user')
  sessionStorage.removeItem('kiviq_pass')
}

export function generateToken(bytes = 32) {
  const arr = new Uint8Array(bytes)
  crypto.getRandomValues(arr)
  return Array.from(arr, b => b.toString(16).padStart(2, '0')).join('')
}

// generateId returns an opaque, unique agent ID. crypto.randomUUID is available
// in secure contexts (the dashboard is served over HTTPS).
export function generateId() {
  return crypto.randomUUID()
}

export function getAuth() {
  const { user, pass } = getCredentials()
  return btoa(`${user}:${pass}`)
}

// apiUrl resolves an API path against the document base. The dashboard is served
// either at the site root (direct access) or under a reverse-proxy sub-path
// (Home Assistant ingress injects a <base> tag pointing at its prefix), so paths
// must resolve relative to document.baseURI rather than absolute from "/".
export function apiUrl(path) {
  return new URL(path.replace(/^\//, ''), document.baseURI).toString()
}

// wsUrl is apiUrl's WebSocket sibling: same base resolution, but it swaps the
// scheme to ws/wss to match the page's http/https.
export function wsUrl(path) {
  const u = new URL(path.replace(/^\//, ''), document.baseURI)
  u.protocol = u.protocol === 'https:' ? 'wss:' : 'ws:'
  return u
}

export function formatBytes(b) {
  if (!b) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0, v = b
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
  return v.toFixed(1) + ' ' + units[i]
}

export function formatRate(bytesPerSec) {
  if (bytesPerSec < 1024) return bytesPerSec.toFixed(0) + ' B/s'
  if (bytesPerSec < 1048576) return (bytesPerSec / 1024).toFixed(1) + ' KB/s'
  if (bytesPerSec < 1073741824) return (bytesPerSec / 1048576).toFixed(1) + ' MB/s'
  return (bytesPerSec / 1073741824).toFixed(2) + ' GB/s'
}

export function textColor(v) {
  return v > 90 ? 'text-danger' : v > 70 ? 'text-warning' : 'text-success'
}

export function barColor(v) {
  return v > 90 ? 'bg-danger' : v > 70 ? 'bg-warning' : 'bg-success'
}

export function tempColor(c) {
  if (c >= 90) return 'text-danger'
  if (c >= 70) return 'text-warning'
  return 'text-success'
}

// formatLastSeen renders an RFC3339 timestamp as a relative age ("4s ago",
// "2m ago", "1h ago", "3d ago") against now (ms). Both arguments are absolute
// instants, so the result is timezone-independent.
export function formatLastSeen(lastSeenIso, now) {
  if (!lastSeenIso) return 'never'
  const then = Date.parse(lastSeenIso)
  if (Number.isNaN(then)) return 'unknown'
  const s = Math.max(0, Math.floor((now - then) / 1000))
  if (s < 5) return 'just now'
  if (s < 60) return `${s}s ago`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ago`
  return `${Math.floor(h / 24)}d ago`
}

export function formatUptime(seconds) {
  const s = seconds || 0
  const d = Math.floor(s / 86400)
  const h = Math.floor((s % 86400) / 3600)
  const m = Math.floor((s % 3600) / 60)
  if (d > 0) return `${d}d ${h}h`
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}
