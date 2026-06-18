<template>
  <section>
    <h3 class="text-xs font-bold text-ink mb-2">
      Memory
    </h3>
    <div class="grid grid-cols-2 md:grid-cols-4 gap-x-4 gap-y-1 text-sm mb-2">
      <div>
        <span class="text-mute text-xs">Total </span>
        <span class="text-ink font-mono tabular-nums">{{ formatBytes(agent.memory?.total_bytes) }}</span>
      </div>
      <div>
        <span class="text-mute text-xs">Used </span>
        <span class="text-ink font-mono tabular-nums">{{ formatBytes(agent.memory?.used_bytes) }}</span>
      </div>
      <div>
        <span class="text-mute text-xs">Free </span>
        <span class="text-ink font-mono tabular-nums">{{ formatBytes(agent.memory?.free_bytes) }}</span>
      </div>
      <div>
        <span class="text-mute text-xs">Usage </span>
        <span
          class="font-mono tabular-nums"
          :class="textColor(agent.memory?.usage_percent)"
        >{{ (agent.memory?.usage_percent || 0).toFixed(1) }}%</span>
      </div>
    </div>
    <StatBar
      label="RAM"
      :value="agent.memory?.usage_percent || 0"
    />
    <div
      v-if="agent.memory?.swap_total"
      class="mt-2"
    >
      <div class="flex gap-4 text-xs mb-1">
        <span class="text-mute">Swap {{ formatBytes(agent.memory.swap_used) }} / {{ formatBytes(agent.memory.swap_total) }}</span>
      </div>
      <StatBar
        label="Swap"
        :value="agent.memory.swap_total ? (agent.memory.swap_used / agent.memory.swap_total * 100) : 0"
      />
    </div>
    <div class="mt-3">
      <StatChart
        :agent-id="agent.agent_id"
        label="Memory over time"
        field="mem"
        color="#8b5cf6"
      />
    </div>
  </section>
</template>

<script setup>
import StatBar from './StatBar.vue'
import StatChart from './StatChart.vue'
import { formatBytes, textColor } from '../utils.js'

defineProps({
  agent: Object,
})
</script>
