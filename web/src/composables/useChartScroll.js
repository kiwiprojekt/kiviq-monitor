import { onMounted, onUnmounted } from 'vue'

// Cap the scroll animation at ~30fps: the window moves ~1px/sec, so this is
// smooth to the eye while drawing a fraction of the frames a raw rAF loop would.
const SCROLL_FRAME_MS = 33

// useChartScroll owns a canvas's redraw cadence and lifecycle, given a draw()
// callback. It runs a continuous ~30fps loop that slides the time window left
// while the tab is visible and motion is allowed, and exposes scheduleDraw()
// for coalesced on-demand repaints (new data, window switch, resize, tooltip).
// It wires its own mount/unmount hooks: starts on mount, and on unmount stops
// the loop, cancels any pending frame, and drops the visibility listener.
export function useChartScroll(draw) {
  // Two independent frame handles. The scroll loop and the coalesced one-shot
  // repaint have separate lifecycles; sharing a single handle let a one-shot
  // captured while the tab was hidden block the loop from relaunching on the
  // way back to visible (start() would see the pending frame and decline).
  let loopRaf = null
  let pendingRaf = null
  let animating = false
  let lastDrawTime = 0
  let reducedMotion = false

  // Coalesce discrete pixel-changing events into a single trailing-edge draw,
  // so a burst within one frame paints once. While the scroll loop is running
  // this short-circuits — the loop will redraw within a frame anyway.
  function scheduleDraw() {
    if (pendingRaf !== null || loopRaf !== null) return
    pendingRaf = requestAnimationFrame(() => {
      pendingRaf = null
      draw()
    })
  }

  function scrollLoop(ts) {
    if (!animating) {
      loopRaf = null
      return
    }
    if (ts - lastDrawTime >= SCROLL_FRAME_MS) {
      lastDrawTime = ts
      draw()
    }
    loopRaf = requestAnimationFrame(scrollLoop)
  }

  function start() {
    if (animating || reducedMotion || document.hidden) return
    animating = true
    if (loopRaf === null) loopRaf = requestAnimationFrame(scrollLoop)
  }

  function stop() {
    animating = false
  }

  function onVisibilityChange() {
    if (document.hidden) {
      stop()
    } else {
      start()
      scheduleDraw()
    }
  }

  onMounted(() => {
    reducedMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false
    document.addEventListener('visibilitychange', onVisibilityChange)
    start()
  })

  onUnmounted(() => {
    stop()
    if (loopRaf) cancelAnimationFrame(loopRaf)
    if (pendingRaf) cancelAnimationFrame(pendingRaf)
    document.removeEventListener('visibilitychange', onVisibilityChange)
  })

  return { scheduleDraw }
}
