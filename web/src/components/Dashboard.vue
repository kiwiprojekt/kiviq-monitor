<template>
  <div>
    <template v-if="loading">
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <div
          v-for="i in 3"
          :key="i"
          class="bg-surface-soft border border-hairline p-4 animate-pulse"
        >
          <div class="flex items-center gap-2 mb-3">
            <div class="w-2 h-2 bg-surface-card" />
            <div class="h-4 bg-surface-card w-24" />
            <div class="ml-auto h-3 bg-surface-card w-12" />
          </div>
          <div class="space-y-2.5 mb-3">
            <div class="h-2 bg-surface-card" />
            <div class="h-2 bg-surface-card w-3/4" />
            <div class="h-2 bg-surface-card" />
          </div>
          <div class="space-y-1.5">
            <div class="h-3 bg-surface-card w-full" />
            <div class="h-3 bg-surface-card w-2/3" />
          </div>
        </div>
      </div>
    </template>
    <template v-else-if="agents.length === 0">
      <div class="flex flex-col items-center justify-center py-20 text-center">
        <div class="w-16 h-16 bg-surface-card flex items-center justify-center mb-4">
          <svg class="w-7 h-7 text-ash" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/>
          </svg>
        </div>
        <h3 class="text-sm font-medium text-ink mb-1">
          No agents connected
        </h3>
        <p class="text-sm text-body max-w-xs">
          Start an agent on your server to begin monitoring. Check the documentation for setup instructions.
        </p>
      </div>
    </template>
    <template v-else>
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <AgentCard
          v-for="agent in agents"
          :key="agent.agent_id"
          :agent="agent"
          @click="$emit('select', agent)"
        />
      </div>
    </template>
  </div>
</template>

<script setup>
import AgentCard from './AgentCard.vue'

defineProps({ agents: Array, loading: Boolean })
defineEmits(['select'])
</script>
