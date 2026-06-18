<template>
  <div class="bg-surface-soft p-3">
    <div class="flex items-center justify-between mb-2">
      <div
        class="text-lg font-bold tabular-nums font-mono leading-none"
        :class="stale ? 'text-mute' : 'text-ink'"
      >
        {{ stale ? '' : kpiPrefix }}{{ displayKpi }}
      </div>
      <div class="flex gap-0.5 bg-surface-card p-0.5">
        <button
          v-for="w in windows"
          :key="w.ms"
          class="px-2 py-0.5 text-xs transition-colors cursor-pointer"
          :class="selectedWindow === w.ms ? 'bg-ink text-on-dark' : 'text-mute hover:text-ink'"
          @click="setWindow(w.ms)"
        >
          {{ w.label }}
        </button>
      </div>
    </div>

    <div class="relative h-28">
      <canvas
        ref="canvas"
        class="w-full h-full cursor-crosshair"
        @mousemove="onMouseMove"
        @mouseleave="onMouseLeave"
      />
      <div
        v-if="tooltip.visible"
        class="absolute pointer-events-none bg-ink border border-ink px-2 py-1 text-xs z-10"
        :style="{ left: tooltip.x + 'px', top: tooltip.y + 'px', transform: 'translate(-50%, -100%) translateY(-8px)' }"
      >
        <div class="text-on-dark font-bold tabular-nums font-mono">
          {{ tooltip.value }}
        </div>
        <div class="text-on-dark-mute text-[10px]">
          {{ tooltip.time }}
        </div>
      </div>
    </div>

    <div class="flex justify-between text-[10px] text-stone mt-1 font-mono tabular-nums">
      <span>{{ pointCount }} pts · {{ resolution }}s</span>
      <span>{{ timeRange }}</span>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted, onUnmounted } from 'vue'
import { formatRate } from '../utils.js'
import { computeRates, interpolateAt, sampleSeriesAt, mergeLivePoint } from '../composables/useChartProcessing.js'
import { useChartScroll } from '../composables/useChartScroll.js'
import { useWebSocket, OFFLINE_THRESHOLD_MS } from '../composables/useWebSocket.js'

const props = defineProps({
  agentId: String,
  label: String,
  kpiPrefix: { type: String, default: '' },
  field: String,
  color: { type: String, default: '#007aff' },
  unit: { type: String, default: '%' },
  maxVal: { type: Number, default: 100 },
  computeRate: { type: Boolean, default: false },
})

// The shared realtime singleton is reached directly — history is no longer
// drilled down through the section components as props.
const { state, getHistory, onHistoryUpdate } = useWebSocket()

const windows = [
  { label: '1m', ms: 60000 },
  { label: '5m', ms: 300000 },
  { label: '15m', ms: 900000 },
  { label: '1h', ms: 3600000 },
]

const selectedWindow = ref(300000)
const canvas = ref(null)
const kpiValue = ref('—')
const pointCount = ref(0)
const resolution = ref(5)
const timeRange = ref('—')

// Timestamp (ms) of the most recent raw point. When it ages past the offline
// threshold the agent has gone silent, so the KPI reads "—" instead of a stale
// last value — mirroring the blank right side the chart draws.
const lastPointT = ref(0)
const stale = computed(() => !lastPointT.value || state.now - lastPointT.value > OFFLINE_THRESHOLD_MS)
const displayKpi = computed(() => (stale.value ? '—' : kpiValue.value))

const tooltip = reactive({ visible: false, x: 0, y: 0, value: '', time: '' })

let rawPoints = []
let processedPoints = []
let unsubscribe = null
let resizeObserver = null
let canvasW = 0
let canvasH = 0

function setWindow(ms) {
  selectedWindow.value = ms
  rebuildProcessed()
}

function rebuildProcessed() {
  if (props.computeRate) {
    processedPoints = computeRates(rawPoints, props.field)
    if (processedPoints.length > 0) {
      kpiValue.value = formatRate(processedPoints[processedPoints.length - 1].val)
    } else {
      kpiValue.value = '—'
    }
  } else {
    processedPoints = rawPoints.map(p => ({ t: p.t, val: p[props.field] || 0 }))
    if (processedPoints.length > 0) {
      kpiValue.value = processedPoints[processedPoints.length - 1].val.toFixed(1) + props.unit
    } else {
      kpiValue.value = '—'
    }
  }
  pointCount.value = processedPoints.length
  lastPointT.value = rawPoints.length ? rawPoints[rawPoints.length - 1].t : 0

  if (rawPoints.length >= 2) {
    resolution.value = Math.round((rawPoints[1].t - rawPoints[0].t) / 1000)
  }

  const windowMs = selectedWindow.value
  if (windowMs >= 3600000) timeRange.value = '1 hour'
  else if (windowMs >= 900000) timeRange.value = '15 min'
  else if (windowMs >= 300000) timeRange.value = '5 min'
  else timeRange.value = '1 min'

  scheduleDraw()
}

function onNewPoint(pt) {
  if (!pt) return
  mergeLivePoint(rawPoints, pt, Date.now() - windows[windows.length - 1].ms)
  rebuildProcessed()
}

function sizeCanvas() {
  const c = canvas.value
  if (!c) return
  const rect = c.getBoundingClientRect()
  if (c.width === rect.width * 2 && c.height === rect.height * 2) return
  c.width = rect.width * 2
  c.height = rect.height * 2
  canvasW = rect.width
  canvasH = rect.height
}

function formatTooltipValue(v) {
  if (props.computeRate) return formatRate(v)
  return v.toFixed(1) + props.unit
}

function onMouseMove(e) {
  const rect = canvas.value?.getBoundingClientRect()
  if (!rect) return
  const x = e.clientX - rect.left
  const windowMs = selectedWindow.value
  const now = Date.now()
  const windowEnd = now + 1000
  const windowStart = windowEnd - windowMs
  const t = windowStart + (x / canvasW) * windowMs
  const val = interpolateAt(processedPoints, t)
  if (val === null) { tooltip.visible = false; return }

  const date = new Date(t)
  const timeStr = date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })

  tooltip.visible = true
  tooltip.x = x
  tooltip.y = 0
  tooltip.value = formatTooltipValue(val)
  tooltip.time = timeStr
  scheduleDraw()
}

function onMouseLeave() {
  tooltip.visible = false
  scheduleDraw()
}

function drawChart() {
  const c = canvas.value
  if (!c) return
  sizeCanvas()
  const ctx = c.getContext('2d')
  if (!ctx) return
  ctx.setTransform(2, 0, 0, 2, 0, 0)

  const w = canvasW
  const h = canvasH

  ctx.clearRect(0, 0, w, h)

  ctx.strokeStyle = 'rgba(15,0,0,0.08)'
  ctx.lineWidth = 0.5

  let max
  if (props.computeRate) {
    max = 1
    for (const d of processedPoints) {
      if (d.val > max) max = d.val
    }
    max *= 1.1
  } else {
    max = props.maxVal
  }

  for (let i = 0; i <= 4; i++) {
    const y = (h / 4) * i
    ctx.beginPath()
    ctx.moveTo(0, y)
    ctx.lineTo(w, y)
    ctx.stroke()

    const val = max * (1 - i / 4)
    ctx.fillStyle = 'rgba(100,98,98,0.5)'
    ctx.font = '9px ui-monospace, monospace'
    ctx.textAlign = 'right'
    ctx.textBaseline = 'middle'
    const label = props.computeRate ? formatRate(val) : val.toFixed(0) + '%'
    ctx.fillText(label, w - 2, y === 0 ? 6 : y)
  }

  if (processedPoints.length === 0) {
    ctx.fillStyle = 'rgba(100,98,98,0.5)'
    ctx.font = '11px ui-monospace, monospace'
    ctx.textAlign = 'center'
    ctx.fillText('Waiting for data...', w / 2, h / 2)
    return
  }

  const windowMs = selectedWindow.value
  const now = Date.now()
  const windowEnd = now + 1000
  const windowStart = windowEnd - windowMs

  function valToY(v) {
    return h - (v / max) * h
  }

  const STEPS = Math.max(w * 2, 200)
  const stepMs = windowMs / STEPS

  // Draw only up to the last real data point. Past it (a silent agent, or just
  // the few seconds between the last report and now) the right side stays blank
  // instead of flat-extending the last value.
  const lastT = processedPoints[processedPoints.length - 1].t

  // Sample every step in a single forward walk rather than scanning the series
  // from the start for each of ~1000 steps every frame.
  const samples = sampleSeriesAt(processedPoints, windowStart, stepMs, STEPS)

  ctx.beginPath()
  let started = false
  let lastDrawnX = 0
  for (let i = 0; i <= STEPS; i++) {
    const t = windowStart + i * stepMs
    if (t > lastT) break
    const val = samples[i]
    if (val === null) continue
    const x = (i / STEPS) * w
    const y = valToY(val)
    if (!started) {
      ctx.moveTo(x, y)
      started = true
    } else {
      ctx.lineTo(x, y)
    }
    lastDrawnX = x
  }
  if (started) {
    ctx.lineTo(lastDrawnX, h)
    ctx.lineTo(0, h)
    ctx.closePath()
    ctx.fillStyle = props.color + '15'
    ctx.fill()
    ctx.strokeStyle = props.color
    ctx.lineWidth = 1.5
    ctx.stroke()
  }

  if (tooltip.visible) {
    const t = windowStart + (tooltip.x / w) * windowMs
    const val = interpolateAt(processedPoints, t)
    if (val !== null) {
      const x = ((t - windowStart) / windowMs) * w
      const y = valToY(val)
      ctx.beginPath()
      ctx.arc(x, y, 3, 0, Math.PI * 2)
      ctx.fillStyle = props.color
      ctx.fill()
      ctx.strokeStyle = '#fdfcfc'
      ctx.lineWidth = 1.5
      ctx.stroke()

      ctx.setLineDash([3, 3])
      ctx.strokeStyle = 'rgba(100,98,98,0.3)'
      ctx.lineWidth = 1
      ctx.beginPath()
      ctx.moveTo(x, 0)
      ctx.lineTo(x, h)
      ctx.stroke()
      ctx.setLineDash([])
    }
  }
}

// The scrolling redraw loop, visibility/reduced-motion gating, and coalesced
// on-demand repaint scheduling all live in a dedicated composable; the
// component just hands it the draw function.
const { scheduleDraw } = useChartScroll(drawChart)

// Keep the trailing blank edge growing and the KPI flipping to "—" even when the
// scroll loop isn't running (reduced motion, or hidden tab just made visible).
watch(() => state.now, () => scheduleDraw())

onMounted(() => {
  sizeCanvas()
  resizeObserver = new ResizeObserver(() => { sizeCanvas(); scheduleDraw() })
  if (canvas.value?.parentElement) resizeObserver.observe(canvas.value.parentElement)

  if (props.agentId) {
    unsubscribe = onHistoryUpdate(props.agentId, onNewPoint)
    getHistory(props.agentId).then(raw => {
      const live = rawPoints
      rawPoints = raw
      const cutoff = Date.now() - windows[windows.length - 1].ms
      for (const pt of live) mergeLivePoint(rawPoints, pt, cutoff)
      rebuildProcessed()
    })
  }

  // Initial paint; the scroll loop and on-demand repaints are owned by useChartScroll.
  scheduleDraw()
})

onUnmounted(() => {
  if (unsubscribe) unsubscribe()
  if (resizeObserver) resizeObserver.disconnect()
})
</script>
