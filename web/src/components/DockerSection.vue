<template>
  <section>
    <div
      v-if="!(agent.docker?.containers?.length)"
      class="flex flex-col items-center justify-center py-10 text-center"
    >
      <div class="w-12 h-12 bg-surface-card flex items-center justify-center mb-3">
        <span class="text-ash text-lg font-bold">[-]</span>
      </div>
      <p class="text-sm text-body">
        No containers running
      </p>
    </div>
    <div
      v-else
      class="space-y-2"
    >
      <DockerContainerRow
        v-for="c in (agent.docker?.containers || [])"
        :key="c.id"
        :container="c"
        :agent-ip="agentIp"
      />
    </div>
  </section>
</template>

<script setup>
import { computed } from 'vue'
import DockerContainerRow from './DockerContainerRow.vue'

const props = defineProps({ agent: Object })

// Best-effort host for port links: the first non-loopback IPv4 the agent
// reported (NetworkInfo.ip is a comma-joined list). On a multi-homed host this
// may not be the address actually reachable from the browser.
const agentIp = computed(() => {
  if (!props.agent?.network) return ''
  for (const n of props.agent.network) {
    if (n.ip) {
      const ip = n.ip.split(',')[0].trim()
      if (ip) return ip
    }
  }
  return ''
})
</script>
