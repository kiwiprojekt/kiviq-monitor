<template>
  <div
    class="bg-surface-soft border border-hairline transition-colors cursor-pointer"
    :class="expanded ? 'bg-surface-card' : 'hover:bg-surface-card'"
    @click="expanded = !expanded"
  >
    <div class="p-3">
      <div class="flex items-center justify-between gap-2">
        <div class="flex items-center gap-2 min-w-0">
          <span
            class="w-1.5 h-1.5 rounded-full shrink-0"
            :class="container.state === 'running' ? 'bg-success' : 'bg-ash'"
          />
          <span class="font-bold text-ink text-sm truncate">{{ displayName }}</span>
          <span class="text-xs text-stone font-mono shrink-0">{{ container.state }}</span>
        </div>
        <svg
          class="w-3.5 h-3.5 text-mute shrink-0 transition-transform duration-150"
          :class="expanded ? 'rotate-180' : ''"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M6 9l6 6 6-6" />
        </svg>
      </div>
      <div class="flex gap-4 mt-1.5 text-xs text-body font-mono tabular-nums">
        <span>CPU {{ container.cpu_percent.toFixed(1) }}%</span>
        <span>RAM {{ formatBytes(container.memory_usage_bytes) }}</span>
      </div>
    </div>

    <div
      v-if="expanded"
      class="px-3 pb-3 border-t border-hairline"
    >
      <div class="pt-3 grid grid-cols-2 gap-3 mb-3">
        <div class="bg-surface-soft p-2">
          <div class="text-[10px] text-mute mb-1 font-mono">CPU</div>
          <canvas ref="sparkCanvas" class="w-full block" style="height: 36px;" />
        </div>
        <div class="bg-surface-soft p-2">
          <div class="text-[10px] text-mute mb-1 font-mono">RAM</div>
          <canvas ref="sparkCanvas2" class="w-full block" style="height: 36px;" />
        </div>
      </div>
      <div class="grid grid-cols-2 gap-x-4 gap-y-1.5 text-xs">
        <div>
          <span class="text-mute">Image </span>
          <span class="text-body font-mono">{{ container.image }}</span>
        </div>
        <div>
          <span class="text-mute">ID </span>
          <span class="text-body font-mono">{{ container.id?.slice(0, 12) }}</span>
        </div>
        <div>
          <span class="text-mute">CPU </span>
          <span class="text-body font-mono tabular-nums">{{ container.cpu_percent.toFixed(1) }}%</span>
        </div>
        <div>
          <span class="text-mute">RAM </span>
          <span class="text-body font-mono tabular-nums">{{ formatBytes(container.memory_usage_bytes) }} / {{ formatBytes(container.memory_limit_bytes) }}</span>
        </div>
        <div>
          <span class="text-mute">Net RX </span>
          <span class="text-body font-mono tabular-nums">{{ formatBytes(container.network_rx_bytes) }}</span>
        </div>
        <div>
          <span class="text-mute">Net TX </span>
          <span class="text-body font-mono tabular-nums">{{ formatBytes(container.network_tx_bytes) }}</span>
        </div>
        <div class="col-span-2">
          <span class="text-mute">Ports </span>
          <span v-if="!boundPorts.length" class="text-body font-mono">none</span>
          <span v-else class="text-body font-mono">
            <template v-for="(p, idx) in boundPorts" :key="idx">
              <a
                :href="p.url"
                target="_blank"
                rel="noopener noreferrer"
                class="text-accent hover:text-accent-hover hover:underline"
                @click.stop
              >{{ p.label }}</a>
              <span v-if="idx < boundPorts.length - 1">, </span>
            </template>
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import { formatBytes } from '../utils.js'

const props = defineProps({
  container: Object,
  agentIp: String
})

const expanded = ref(false)
const sparkCanvas = ref(null)
const sparkCanvas2 = ref(null)

const displayName = computed(() =>
  (props.container.name || props.container.id || '').replace(/^\//, '')
)

const MAX_HISTORY = 120
const history = []

function drawSpark(canvas, getData, color) {
  if (!canvas) return
  const dpr = window.devicePixelRatio || 1
  const rect = canvas.getBoundingClientRect()
  if (!rect.width) return
  canvas.width = rect.width * dpr
  canvas.height = rect.height * dpr
  const ctx = canvas.getContext('2d')
  ctx.scale(dpr, dpr)
  const w = rect.width
  const h = rect.height
  ctx.clearRect(0, 0, w, h)
  if (history.length < 2) return
  ctx.beginPath()
  history.forEach((pt, i) => {
    const x = (i / (history.length - 1)) * w
    const y = h - (Math.min(Math.max(getData(pt), 0), 100) / 100) * h
    i === 0 ? ctx.moveTo(x, y) : ctx.lineTo(x, y)
  })
  ctx.lineTo(w, h); ctx.lineTo(0, h); ctx.closePath()
  ctx.fillStyle = color + '20'
  ctx.fill()
  ctx.beginPath()
  history.forEach((pt, i) => {
    const x = (i / (history.length - 1)) * w
    const y = h - (Math.min(Math.max(getData(pt), 0), 100) / 100) * h
    i === 0 ? ctx.moveTo(x, y) : ctx.lineTo(x, y)
  })
  ctx.strokeStyle = color
  ctx.lineWidth = 1.5
  ctx.stroke()
}

function redraw() {
  drawSpark(sparkCanvas.value, p => p.cpu, '#007aff')
  drawSpark(sparkCanvas2.value, p => p.memPct, '#8b5cf6')
}

function pushPoint() {
  const cpu = props.container.cpu_percent || 0
  const memPct = props.container.memory_limit_bytes > 0
    ? (props.container.memory_usage_bytes / props.container.memory_limit_bytes) * 100
    : 0
  history.push({ cpu, memPct })
  if (history.length > MAX_HISTORY) history.shift()
  if (expanded.value) nextTick(redraw)
}

// Use the port's own bound IP, unless it's a wildcard (any-interface) bind,
// in which case fall back to the agent's address, then the page host.
function resolveHost(ip) {
  if (ip && ip !== '0.0.0.0' && ip !== '::') return ip
  return props.agentIp || location.hostname || 'localhost'
}

const boundPorts = computed(() => {
  const ports = props.container.ports || []
  return ports
    .filter(p => p.public_port)
    .map(p => ({
      label: `${p.public_port}->${p.private_port}/${p.type}`,
      url: `http://${resolveHost(p.ip)}:${p.public_port}`
    }))
})

watch(() => props.container, () => pushPoint(), { immediate: true })
watch(expanded, (v) => { if (v) nextTick(redraw) })
</script>
