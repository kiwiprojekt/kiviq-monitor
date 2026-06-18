<template>
  <section>
    <div class="flex items-center justify-between mb-2">
      <h3 class="text-xs font-bold text-ink">
        CPU
      </h3>
      <span
        v-if="agent.cpu?.model_name"
        class="text-xs text-stone font-mono truncate ml-4 text-right"
      >
        {{ agent.cpu.model_name }}
        <template v-if="agent.cpu?.freq_min_mhz || agent.cpu?.freq_max_mhz">
          ·
          <span v-if="agent.cpu.freq_min_mhz">{{ agent.cpu.freq_min_mhz.toFixed(0) }}</span><template v-if="agent.cpu.freq_min_mhz && agent.cpu.freq_max_mhz">–</template><span v-if="agent.cpu.freq_max_mhz">{{ agent.cpu.freq_max_mhz.toFixed(0) }} MHz</span>
        </template>
      </span>
    </div>
    <div class="space-y-1">
      <StatBar
        label="all"
        :value="agent.cpu?.usage_percent || 0"
        label-class="w-10 font-mono"
        value-class="w-12"
      >
        <template
          v-if="agent.cpu?.freq_mhz?.length"
          #trailing
        >
          <span class="w-14 shrink-0" />
        </template>
      </StatBar>
      <StatBar
        v-for="(core, i) in (agent.cpu?.per_core || [])"
        :key="i"
        :label="String(i)"
        :value="core"
        :colored-value="false"
        label-class="w-10 font-mono"
        value-class="text-body w-12"
        bar-height="h-1"
      >
        <template
          v-if="agent.cpu?.freq_mhz?.[i]"
          #trailing
        >
          <span class="text-xs text-stone w-14 text-right tabular-nums font-mono">{{ agent.cpu.freq_mhz[i].toFixed(0) }}</span>
        </template>
      </StatBar>
    </div>
    <div
      v-if="agent.cpu?.temperatures?.length"
      class="flex items-center flex-wrap gap-x-3 gap-y-1 mt-3"
    >
      <span class="text-xs text-mute shrink-0">Temp</span>
      <span
        v-for="temp in agent.cpu.temperatures"
        :key="temp.label"
        class="text-xs font-mono"
      >
        <span class="text-stone">{{ temp.label }}</span>
        <span
          class="tabular-nums ml-1"
          :class="tempColor(temp.celsius)"
        >{{ temp.celsius.toFixed(1) }}&deg;</span>
      </span>
    </div>
    <div class="mt-3">
      <StatChart
        :agent-id="agent.agent_id"
        label="CPU"
        field="cpu"
        color="#007aff"
      />
    </div>
  </section>
</template>

<script setup>
import StatChart from './StatChart.vue'
import StatBar from './StatBar.vue'
import { tempColor } from '../utils.js'

defineProps({
  agent: Object,
})
</script>
