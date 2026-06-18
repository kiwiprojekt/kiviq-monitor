import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { useChartScroll } from './useChartScroll.js'

// A controllable requestAnimationFrame: callbacks queue and only run when the
// test fires them, with a test-driven timestamp. This lets us reproduce the
// hidden→visible race deterministically instead of relying on real frame timing.
function installFakeRaf() {
  let queue = []
  let nextId = 1
  let nowTs = 0

  globalThis.requestAnimationFrame = (cb) => {
    const id = nextId++
    queue.push({ id, cb })
    return id
  }
  globalThis.cancelAnimationFrame = (id) => {
    queue = queue.filter((f) => f.id !== id)
  }

  return {
    // Fire exactly the frames queued right now (not ones they enqueue).
    fireAll() {
      const current = queue
      queue = []
      for (const f of current) f.cb(nowTs)
    },
    // Advance the clock and fire one generation per frame, n times.
    runFrames(n, dt) {
      for (let i = 0; i < n; i++) {
        nowTs += dt
        const current = queue
        queue = []
        for (const f of current) f.cb(nowTs)
      }
    },
  }
}

function setHidden(hidden) {
  Object.defineProperty(document, 'hidden', { value: hidden, configurable: true })
  document.dispatchEvent(new Event('visibilitychange'))
}

function mountScroll() {
  let draws = 0
  const Harness = {
    template: '<div />',
    setup() {
      const { scheduleDraw } = useChartScroll(() => { draws++ })
      return { scheduleDraw }
    },
  }
  const wrapper = mount(Harness)
  return { wrapper, scheduleDraw: () => wrapper.vm.scheduleDraw(), getDraws: () => draws }
}

describe('useChartScroll', () => {
  let raf
  beforeEach(() => {
    raf = installFakeRaf()
    Object.defineProperty(document, 'hidden', { value: false, configurable: true })
  })
  afterEach(() => {
    delete globalThis.requestAnimationFrame
    delete globalThis.cancelAnimationFrame
  })

  // Regression: a coalesced scheduleDraw() captured while the tab is hidden must
  // not prevent the continuous scroll loop from resuming when the tab becomes
  // visible again. Previously both shared one rafId, so start() saw the pending
  // one-shot, declined to launch the loop, and scrolling silently froze at
  // ~1 repaint/sec until the next visibility toggle.
  it('resumes the continuous scroll loop after a hidden→visible cycle with a pending repaint', () => {
    const { scheduleDraw, getDraws } = mountScroll()

    // Tab hidden: the loop stops, and the in-flight loop frame completes.
    setHidden(true)
    raf.fireAll()

    // A repaint is requested while hidden (e.g. the once-a-second clock watcher).
    scheduleDraw()

    // Tab visible again.
    setHidden(false)

    const before = getDraws()
    raf.runFrames(4, 33)
    // The loop must draw on multiple successive frames, not stall after one.
    expect(getDraws() - before).toBeGreaterThanOrEqual(3)
  })

  it('draws continuously on every eligible frame while visible', () => {
    const { getDraws } = mountScroll()
    const before = getDraws()
    raf.runFrames(4, 33)
    expect(getDraws() - before).toBeGreaterThanOrEqual(3)
  })
})
