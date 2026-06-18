<template>
  <div class="min-h-screen bg-canvas flex">
    <AppSidebar
      :agents="agents"
      :selected-id="selectedAgentId"
      :connected="state.connected"
      :dashboard-active="!selectedAgentId && view === 'dashboard'"
      :settings-active="view === 'settings'"
      @select="onSelectAgent"
      @logout="logout"
      @settings="selectedAgentId = null; view = 'settings'"
      @dashboard="selectedAgentId = null; view = 'dashboard'"
    />

    <div class="flex-1 flex flex-col min-w-0 h-screen overflow-y-auto" style="scrollbar-gutter: stable">
      <header class="sticky top-0 z-10 bg-canvas border-b border-hairline px-6 py-3">
        <div class="flex items-center justify-between min-h-9">
          <div class="flex items-center gap-3">
            <template v-if="selectedAgent">
              <button
                class="text-mute hover:text-ink transition-colors p-1 -ml-1 rounded cursor-pointer"
                aria-label="Back to dashboard"
                @click="selectedAgentId = null"
              >
                <svg
                  class="w-5 h-5"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                >
                  <path d="M19 12H5M12 19l-7-7 7-7" />
                </svg>
              </button>
              <div class="flex flex-col">
                <h1 class="text-base font-bold text-ink leading-tight">
                  {{ selectedAgent.agent_name || selectedAgent.hostname }}
                </h1>
                <span
                  v-if="selectedOnline"
                  class="text-xs font-medium text-success"
                >● ONLINE</span>
                <span
                  v-else
                  class="text-xs font-medium text-mute"
                  :title="lastSeenAbsolute"
                >● OFFLINE · last seen {{ lastSeenRelative }}</span>
              </div>
            </template>
            <template v-else-if="view === 'settings'">
              <button
                class="text-mute hover:text-ink transition-colors p-1 -ml-1 rounded cursor-pointer"
                aria-label="Back to dashboard"
                @click="view = 'dashboard'"
              >
                <svg
                  class="w-5 h-5"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                >
                  <path d="M19 12H5M12 19l-7-7 7-7" />
                </svg>
              </button>
              <h1 class="text-base font-bold text-ink">
                Settings
              </h1>
            </template>
            <template v-else>
              <h1 class="text-base font-bold text-ink">
                Dashboard
              </h1>
            </template>
          </div>
          <div class="flex items-center gap-2 text-xs text-mute">
            <span
              v-if="selectedAgent"
              class="font-mono"
            >{{ selectedAgent.hostname }}</span>
          </div>
        </div>
      </header>

      <main class="flex-1 px-6 py-5">
        <AdminPanel
          v-if="view === 'settings'"
        />
        <Dashboard
          v-else-if="!selectedAgent"
          :agents="agents"
          :loading="loading"
          @select="onSelectAgent"
        />
        <AgentDetail
          v-else
          :agent="selectedAgent"
        />
      </main>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { clearCredentials, formatLastSeen } from '../utils.js'
import { useWebSocket, useRealtimeConnection, isAgentOnline } from '../composables/useWebSocket.js'
import AppSidebar from './AppSidebar.vue'
import Dashboard from './Dashboard.vue'
import AgentDetail from './AgentDetail.vue'
import AdminPanel from './AdminPanel.vue'

// Own the singleton connection here, at the app root; children reach shared
// state via useWebSocket() directly.
useRealtimeConnection()
const { state, fetchAgents } = useWebSocket()
const selectedAgentId = ref(null)
const loading = ref(true)
const view = ref('dashboard')

const agents = computed(() => Array.from(state.agents.values()))
const selectedAgent = computed(() =>
  selectedAgentId.value ? state.agents.get(selectedAgentId.value) ?? null : null,
)
const selectedOnline = computed(() => isAgentOnline(selectedAgent.value, state.now))
const lastSeenRelative = computed(() => formatLastSeen(selectedAgent.value?.last_seen, state.now))
const lastSeenAbsolute = computed(() =>
  selectedAgent.value?.last_seen ? new Date(selectedAgent.value.last_seen).toLocaleString() : '',
)

onMounted(async () => {
  await fetchAgents()
  loading.value = false
})

function onSelectAgent(agent) {
  if (typeof agent === 'string') {
    selectedAgentId.value = agent
  } else {
    selectedAgentId.value = agent.agent_id
  }
  view.value = 'detail'
}

function logout() {
  clearCredentials()
  location.reload()
}
</script>
