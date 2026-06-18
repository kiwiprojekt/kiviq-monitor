<template>
  <div class="bg-canvas border border-hairline" :class="{ 'grayscale opacity-60': !online }">
    <div class="px-4 border-b border-hairline">
      <Tabs :tabs="tabs">
        <template #host>
          <HostSection :agent="agent" />
        </template>
        <template #docker>
          <DockerSection :agent="agent" />
        </template>
      </Tabs>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import Tabs from './Tabs.vue'
import HostSection from './HostSection.vue'
import DockerSection from './DockerSection.vue'
import { useWebSocket, isAgentOnline } from '../composables/useWebSocket.js'

const props = defineProps({ agent: Object })

const { state } = useWebSocket()
const online = computed(() => isAgentOnline(props.agent, state.now))

const tabs = [
  { id: 'host', label: 'Host' },
  { id: 'docker', label: 'Docker' },
]
</script>
