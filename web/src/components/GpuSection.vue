<template>
  <section>
    <h3 class="text-xs font-bold text-ink mb-2">
      GPU
    </h3>
    <div class="grid grid-cols-2 md:grid-cols-4 gap-x-4 gap-y-1 text-sm mb-2">
      <div>
        <span class="text-mute text-xs">Model </span>
        <span class="text-ink">{{ agent.gpu.name }}</span>
      </div>
      <div>
        <span class="text-mute text-xs">Driver </span>
        <span class="text-ink font-mono text-xs">{{ agent.gpu.driver_version }}</span>
      </div>
      <div>
        <span class="text-mute text-xs">Temp </span>
        <span
          class="font-mono tabular-nums"
          :class="tempColor(agent.gpu.temperature_celsius)"
        >{{ agent.gpu.temperature_celsius }}&deg;C</span>
      </div>
      <div>
        <span class="text-mute text-xs">Power </span>
        <span class="text-ink font-mono tabular-nums">{{ agent.gpu.power_draw_watts }}W</span>
      </div>
    </div>
    <StatBar
      label="GPU"
      :value="agent.gpu.utilization_percent || 0"
    />
    <div class="flex items-center gap-2 text-xs mt-1.5">
      <span class="text-mute">VRAM</span>
      <span class="text-body font-mono tabular-nums">{{ formatBytes(agent.gpu.memory_used) }}</span>
      <span class="text-ash">/</span>
      <span class="text-body font-mono tabular-nums">{{ formatBytes(agent.gpu.memory_total) }}</span>
    </div>
  </section>
</template>

<script setup>
import StatBar from './StatBar.vue'
import { formatBytes, tempColor } from '../utils.js'

defineProps({ agent: Object })
</script>
