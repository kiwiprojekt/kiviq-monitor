import { describe, it, expect } from 'vitest'
import { isAgentOnline, OFFLINE_THRESHOLD_MS } from './useWebSocket.js'

// isAgentOnline is the canonical liveness check on the client. It must derive
// online-ness from last_seen against a ticking clock, not from any server flag,
// so a silent agent flips offline on its own once reports stop.
describe('isAgentOnline', () => {
  const now = 1_000_000_000_000
  const iso = (ms) => new Date(ms).toISOString()

  it('is offline when last_seen is missing', () => {
    expect(isAgentOnline({}, now)).toBe(false)
    expect(isAgentOnline(null, now)).toBe(false)
    expect(isAgentOnline(undefined, now)).toBe(false)
  })

  it('is online when the last report is within the threshold', () => {
    expect(isAgentOnline({ last_seen: iso(now - 1000) }, now)).toBe(true)
    expect(isAgentOnline({ last_seen: iso(now) }, now)).toBe(true)
  })

  it('is offline once the last report ages past the threshold', () => {
    expect(isAgentOnline({ last_seen: iso(now - OFFLINE_THRESHOLD_MS - 1) }, now)).toBe(false)
  })

  it('is timezone-independent (parses the absolute instant from the offset)', () => {
    // Same instant, expressed with a +02:00 offset rather than Z.
    const withOffset = '2001-09-09T03:46:39+02:00' // == 2001-09-09T01:46:39Z
    const instant = Date.parse(withOffset)
    expect(isAgentOnline({ last_seen: withOffset }, instant + 1000)).toBe(true)
    expect(isAgentOnline({ last_seen: withOffset }, instant + OFFLINE_THRESHOLD_MS + 1)).toBe(false)
  })
})
