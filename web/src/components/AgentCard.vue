<template>
  <div
    class="bg-canvas border border-hairline p-4 cursor-pointer hover:bg-surface-soft transition-colors"
    tabindex="0"
    role="button"
    :aria-label="'View details for ' + (agent.agent_name || agent.hostname)"
    @keydown.enter="$emit('select', agent)"
  >
    <div class="flex items-center justify-between mb-3">
      <div class="flex items-center gap-2">
        <span
          class="w-1.5 h-1.5 rounded-full shrink-0"
          :class="online ? 'bg-success' : 'bg-ash'"
        />
        <span class="font-bold text-ink text-sm">{{ agent.agent_name || agent.hostname }}</span>
        <span
          v-if="!online"
          class="text-[10px] font-medium text-mute uppercase tracking-wide"
        >offline</span>
      </div>
      <span class="text-xs text-mute font-mono">{{ uptimeStr }}</span>
    </div>

    <div :class="{ 'grayscale opacity-60': !online }">
    <div class="space-y-2 mb-3">
      <StatBar
        label="CPU"
        :value="agent.cpu?.usage_percent || 0"
      />
      <div
        v-if="agent.cpu?.model_name"
        class="text-xs text-stone truncate font-mono"
        :title="agent.cpu.model_name"
      >
        {{ agent.cpu.model_name }}
      </div>
      <StatBar
        label="RAM"
        :value="agent.memory?.usage_percent || 0"
      />
    </div>

    <div
      v-if="agent.disk?.length"
      class="mb-3"
    >
      <div class="text-xs text-mute mb-1.5">
        Disks
      </div>
      <div
        v-for="d in agent.disk"
        :key="d.device"
        class="flex items-center justify-between text-xs py-0.5"
      >
        <span class="text-body font-mono truncate">{{ d.device }}</span>
        <span
          class="font-mono tabular-nums"
          :class="textColor(d.usage_percent)"
        >{{ d.usage_percent.toFixed(0) }}%</span>
      </div>
    </div>

    <div
      v-if="agent.network?.length"
      class="mb-3"
    >
      <div class="text-xs text-mute mb-1.5">
        Network
      </div>
      <div
        v-for="n in agent.network"
        :key="n.interface"
        class="flex items-center justify-between text-xs py-0.5"
      >
        <span class="text-body truncate">{{ n.interface }}<template v-if="n.speed_mbps > 0"> · {{ n.speed_mbps }}M</template></span>
        <span class="text-body font-mono tabular-nums whitespace-nowrap">&#8595;{{ formatBytes(n.bytes_in) }} &#8593;{{ formatBytes(n.bytes_out) }}</span>
      </div>
    </div>

    <div
      v-if="agent.docker?.containers?.length"
      class="flex items-center gap-1.5 text-xs text-mute"
    >
      <svg class="w-3 h-3 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <rect x="3" y="3" width="5" height="5"/><rect x="10" y="3" width="5" height="5"/><rect x="17" y="3" width="4" height="5"/><rect x="3" y="10" width="5" height="5"/><rect x="10" y="10" width="5" height="5"/>
      </svg>
      {{ agent.docker.containers.length }} container{{ agent.docker.containers.length !== 1 ? 's' : '' }}
    </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import StatBar from './StatBar.vue'
import { formatBytes, textColor, formatUptime } from '../utils.js'
import { useWebSocket, isAgentOnline } from '../composables/useWebSocket.js'

const props = defineProps({ agent: Object })
defineEmits(['select'])

const { state } = useWebSocket()
const online = computed(() => isAgentOnline(props.agent, state.now))
const uptimeStr = computed(() => formatUptime(props.agent.uptime_seconds))
</script>
