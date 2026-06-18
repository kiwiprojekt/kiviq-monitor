<template>
  <section>
    <h3 class="text-xs font-bold text-ink mb-2">
      System
    </h3>
    <div class="grid grid-cols-2 md:grid-cols-4 gap-x-4 gap-y-2 text-sm">
      <div>
        <span class="text-mute text-xs">OS </span>
        <span class="text-ink capitalize">{{ agent.system?.platform || agent.system?.os || '—' }}</span>
      </div>
      <div>
        <span class="text-mute text-xs">Version </span>
        <span class="text-ink">{{ agent.system?.platform_version || '—' }}</span>
      </div>
      <div>
        <span class="text-mute text-xs">Kernel </span>
        <span class="text-ink font-mono text-xs">{{ agent.system?.kernel || '—' }}</span>
      </div>
      <div>
        <span class="text-mute text-xs">Arch </span>
        <span class="text-ink">{{ agent.system?.arch || '—' }}</span>
      </div>
      <div>
        <span class="text-mute text-xs">Hostname </span>
        <span class="text-ink font-mono text-xs">{{ agent.hostname }}</span>
      </div>
      <div>
        <span class="text-mute text-xs">Uptime </span>
        <span class="text-ink">{{ uptimeStr }}</span>
      </div>
      <div v-if="agent.system?.virt_system">
        <span class="text-mute text-xs">Virt </span>
        <span class="text-ink">{{ agent.system.virt_system }} ({{ agent.system.virt_role }})</span>
      </div>
      <div>
        <span class="text-mute text-xs">Cores </span>
        <span class="text-ink">{{ agent.cpu?.cores || 0 }}</span>
      </div>
      <div v-if="agent.system?.load1 !== undefined">
        <span class="text-mute text-xs">Load </span>
        <span class="text-ink font-mono text-xs">{{ agent.system.load1?.toFixed(2) || '—' }} / {{ agent.system.load5?.toFixed(2) || '—' }} / {{ agent.system.load15?.toFixed(2) || '—' }}</span>
      </div>
      <div v-if="agent.system?.process_count">
        <span class="text-mute text-xs">Procs </span>
        <span class="text-ink">{{ agent.system.process_count }}</span>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed } from 'vue'
import { formatUptime } from '../utils.js'

const props = defineProps({ agent: Object })

const uptimeStr = computed(() => formatUptime(props.agent.uptime_seconds))
</script>
