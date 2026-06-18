<template>
  <div
    class="flex items-center gap-2 group"
    :title="label + ': ' + safeValue.toFixed(1) + '%'"
  >
    <span class="text-xs text-mute shrink-0" :class="labelClass">{{ label }}</span>
    <div class="flex-1 bg-surface-card overflow-hidden" :class="barHeight">
      <div
        class="h-full transition-all duration-300"
        :class="barClass"
        :style="{ width: safeValue + '%' }"
      />
    </div>
    <span
      class="text-xs text-right tabular-nums font-mono"
      :class="[coloredValue ? textClass : '', valueClass]"
    >{{ safeValue.toFixed(1) }}%</span>
    <slot name="trailing" />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { barColor, textColor } from '../utils.js'

const props = defineProps({
  label: String,
  value: Number,
  labelClass: { type: String, default: 'w-14' },
  valueClass: { type: String, default: 'w-14' },
  barHeight: { type: String, default: 'h-1.5' },
  // When false, the bar still reflects the usage threshold but the numeric
  // value is rendered neutrally (via valueClass) — used for per-core rows where
  // only the bar should carry color.
  coloredValue: { type: Boolean, default: true },
})

const safeValue = computed(() => {
  const v = props.value
  if (v == null || isNaN(v) || !isFinite(v)) return 0
  return Math.min(v, 100)
})

const barClass = computed(() => barColor(safeValue.value))
const textClass = computed(() => textColor(safeValue.value))
</script>
