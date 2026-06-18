<template>
  <aside class="w-60 bg-surface-dark border-r border-charcoal flex flex-col h-screen shrink-0">
    <div class="px-4 py-4 border-b border-charcoal">
      <div class="flex items-center gap-2.5">
        <div class="w-8 h-8 rounded bg-charcoal flex items-center justify-center shrink-0">
          <span class="text-kiwi-sage text-xs font-bold font-mono">[+]</span>
        </div>
        <span class="text-sm font-bold text-on-dark tracking-tight">kiviq</span>
      </div>
    </div>

    <div class="px-3 pt-3 pb-1">
      <button
        class="w-full text-left px-3 py-2 rounded text-sm transition-colors cursor-pointer flex items-center gap-2"
        :class="dashboardActive
          ? 'bg-surface-dark-elevated text-on-dark'
          : 'text-on-dark-mute hover:bg-surface-dark-elevated hover:text-on-dark'"
        @click="$emit('dashboard')"
      >
        <svg class="w-4 h-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="3" width="7" height="7" /><rect x="14" y="3" width="7" height="7" /><rect x="14" y="14" width="7" height="7" /><rect x="3" y="14" width="7" height="7" />
        </svg>
        Dashboard
      </button>
    </div>

    <nav
      class="flex-1 overflow-y-auto px-3 space-y-0.5"
      aria-label="Agent list"
    >
      <button
        v-for="agent in agents"
        :key="agent.agent_id"
        class="w-full text-left px-3 py-2.5 rounded text-sm transition-colors cursor-pointer"
        :class="selectedId === agent.agent_id
          ? 'bg-surface-dark-elevated text-on-dark'
          : 'text-on-dark-mute hover:bg-surface-dark-elevated hover:text-on-dark'"
        @click="$emit('select', agent.agent_id)"
      >
        <div class="flex items-center gap-2.5">
          <span
            class="w-1.5 h-1.5 rounded-full shrink-0"
            :class="isAgentOnline(agent, state.now) ? 'bg-kiwi-sage' : 'bg-ash'"
          />
          <span class="truncate font-medium">{{ agent.agent_name || agent.hostname }}</span>
        </div>
        <div
          v-if="isAgentOnline(agent, state.now)"
          class="ml-4 mt-1 flex items-center gap-3 text-xs text-on-dark-mute"
        >
          <span>CPU {{ (agent.cpu?.usage_percent || 0).toFixed(0) }}%</span>
          <span>RAM {{ (agent.memory?.usage_percent || 0).toFixed(0) }}%</span>
        </div>
        <div
          v-else
          class="ml-4 mt-1 text-xs text-on-dark-mute"
        >
          offline · {{ formatLastSeen(agent.last_seen, state.now) }}
        </div>
      </button>
    </nav>

    <div class="px-3 py-3 border-t border-charcoal space-y-1">
      <button
        class="w-full px-3 py-2 text-sm rounded transition-colors cursor-pointer text-left flex items-center gap-2"
        :class="settingsActive
          ? 'bg-surface-dark-elevated text-on-dark'
          : 'text-on-dark-mute hover:text-on-dark hover:bg-surface-dark-elevated'"
        @click="$emit('settings')"
      >
        <svg class="w-4 h-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
        </svg>
        Settings
      </button>
      <button
        class="w-full px-3 py-2 text-sm text-on-dark-mute hover:text-on-dark hover:bg-surface-dark-elevated rounded transition-colors cursor-pointer text-left flex items-center gap-2"
        @click="$emit('logout')"
      >
        <svg class="w-4 h-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" /><polyline points="16 17 21 12 16 7" /><line x1="21" y1="12" x2="9" y2="12" />
        </svg>
        Logout
      </button>
    </div>
  </aside>
</template>

<script setup>
import { formatLastSeen } from '../utils.js'
import { useWebSocket, isAgentOnline } from '../composables/useWebSocket.js'

defineProps({
  agents: { type: Array, default: () => [] },
  selectedId: { type: String, default: null },
  connected: { type: Boolean, default: false },
  dashboardActive: { type: Boolean, default: false },
  settingsActive: { type: Boolean, default: false },
})
defineEmits(['select', 'logout', 'settings', 'dashboard'])

const { state } = useWebSocket()
</script>
