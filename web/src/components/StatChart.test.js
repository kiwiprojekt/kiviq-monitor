import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'

// StatChart reaches the realtime singleton directly via useWebSocket(); mock it
// so the test controls history loading and live-point delivery.
const { getHistoryMock, hist, sharedState } = vi.hoisted(() => ({
  getHistoryMock: vi.fn(() => Promise.resolve([])),
  hist: { cb: null },
  sharedState: { now: Date.now() },
}))

vi.mock('../composables/useWebSocket.js', () => ({
  OFFLINE_THRESHOLD_MS: 30000,
  useWebSocket: () => ({
    state: sharedState,
    getHistory: getHistoryMock,
    onHistoryUpdate: (_id, cb) => { hist.cb = cb; return () => {} },
  }),
}))

import StatChart from './StatChart.vue'

// jsdom has no canvas rendering; stub getContext so drawChart can run and we can
// count how often it actually paints. requestAnimationFrame is driven manually
// with a controllable clock so frame-rate throttling is deterministic.
let drawCalls
let rafQueue
let clock
let hidden

function flushRaf() {
  clock += 100 // advance well past the ~30fps throttle each flush
  const batch = rafQueue
  rafQueue = []
  for (const cb of batch) cb(clock)
}

function setHidden(v) {
  hidden = v
  document.dispatchEvent(new Event('visibilitychange'))
}

beforeEach(() => {
  drawCalls = 0
  rafQueue = []
  clock = 0
  hidden = false
  hist.cb = null
  getHistoryMock.mockReset()
  getHistoryMock.mockResolvedValue([])

  vi.stubGlobal('requestAnimationFrame', (cb) => rafQueue.push(cb))
  vi.stubGlobal('cancelAnimationFrame', () => {})
  Object.defineProperty(document, 'hidden', { configurable: true, get: () => hidden })

  const ctx = new Proxy(
    { clearRect: () => { drawCalls++ } },
    { get: (t, p) => t[p] ?? (() => {}) },
  )
  HTMLCanvasElement.prototype.getContext = () => ctx
  vi.stubGlobal('ResizeObserver', class {
    observe() {}
    disconnect() {}
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

const baseProps = { agentId: 'a1', field: 'cpu' }

// matchMedia is absent in jsdom; the component treats absent as "motion allowed".
function stubReducedMotion(matches) {
  vi.stubGlobal('matchMedia', () => ({ matches, addEventListener() {}, removeEventListener() {} }))
}

describe('StatChart scroll animation', () => {
  it('continuously animates the scrolling window while visible', () => {
    mount(StatChart, { props: baseProps })

    flushRaf()
    const after1 = drawCalls
    flushRaf()
    const after2 = drawCalls
    flushRaf()
    const after3 = drawCalls

    expect(after1).toBeGreaterThan(0)
    // The window slides over time, so each frame repaints — this is a live loop.
    expect(after2).toBeGreaterThan(after1)
    expect(after3).toBeGreaterThan(after2)
  })

  it('stops animating when the tab is hidden and resumes when visible', () => {
    mount(StatChart, { props: baseProps })
    flushRaf()
    flushRaf()
    expect(drawCalls).toBeGreaterThan(0)

    setHidden(true)
    flushRaf() // drains the in-flight frame, which must not reschedule
    const frozen = drawCalls
    flushRaf()
    flushRaf()
    expect(drawCalls).toBe(frozen) // no painting while hidden

    setHidden(false)
    flushRaf()
    flushRaf()
    expect(drawCalls).toBeGreaterThan(frozen) // resumed
  })

  it('does not run a continuous loop under reduced motion, but still redraws on new data', () => {
    stubReducedMotion(true)
    mount(StatChart, { props: baseProps })

    flushRaf()
    const initial = drawCalls
    expect(initial).toBeGreaterThan(0) // initial paint

    // Idle: no continuous animation.
    flushRaf()
    flushRaf()
    expect(drawCalls).toBe(initial)

    // A new point still triggers an on-demand redraw.
    hist.cb({ t: Date.now(), cpu: 42 })
    flushRaf()
    expect(drawCalls).toBe(initial + 1)
  })
})
