<template>
  <div class="space-y-4">
    <div class="bg-canvas border border-hairline">
    <div class="px-4 border-b border-hairline">
      <Tabs :tabs="tabs">
        <template #agents>
          <div class="space-y-4">
            <!-- Add agent (top) -->
            <section>
              <div class="flex flex-col sm:flex-row gap-2">
                <input
                  v-model="newName"
                  type="text"
                  :class="[fieldClass, 'flex-1']"
                  placeholder="Display name"
                  @keydown.enter="addAgent"
                >
                <button
                  class="inline-flex items-center justify-center gap-1.5 px-3 py-1.5 bg-ink hover:bg-ink-deep text-on-dark text-xs font-medium rounded transition-colors cursor-pointer shrink-0 disabled:opacity-40 disabled:cursor-not-allowed"
                  :disabled="!newName.trim()"
                  @click="addAgent"
                >
                  <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 5v14M5 12h14"/></svg>
                  Add agent
                </button>
              </div>
              <p class="mt-1.5 text-xs text-mute">A unique ID is generated automatically — the agent's token is its identity.</p>
              <p class="mt-1 text-xs text-ash">A new agent stays hidden from the dashboard until it connects for the first time.</p>
            </section>

            <!-- Agent list -->
            <div
              v-if="agents.length === 0"
              class="border border-hairline border-dashed rounded py-10 text-center"
            >
              <p class="text-sm text-mute">No agents configured yet.</p>
              <p class="text-xs text-ash mt-1">Add one above to get started.</p>
            </div>

            <ul
              v-else
              class="space-y-2"
            >
              <li
                v-for="(agent, idx) in agents"
                :key="agent.id"
                class="border border-hairline bg-surface-soft rounded overflow-hidden"
              >
                <!-- Collapsed header -->
                <div class="flex items-stretch">
                  <button
                    class="flex-1 flex items-center gap-3 px-3 py-2.5 text-left cursor-pointer min-w-0"
                    :aria-expanded="!!expanded[agent.id]"
                    :title="expanded[agent.id] ? 'Collapse' : 'Expand'"
                    @click="toggle(agent.id)"
                  >
                    <svg
                      class="w-4 h-4 shrink-0 text-mute transition-transform"
                      :class="expanded[agent.id] ? 'rotate-90' : ''"
                      viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"
                    ><path d="M9 18l6-6-6-6"/></svg>
                    <span class="min-w-0">
                      <span class="block text-sm font-medium text-ink truncate">{{ agent.name || agent.id }}</span>
                    </span>
                  </button>

                  <!-- Reorder controls -->
                  <div class="flex flex-col justify-center gap-0.5 pr-2 shrink-0">
                    <button
                      class="text-mute hover:text-ink transition-colors cursor-pointer disabled:opacity-20 disabled:cursor-not-allowed"
                      :disabled="idx === 0"
                      aria-label="Move up"
                      title="Move up"
                      @click="moveUp(idx)"
                    >
                      <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M18 15l-6-6-6 6"/></svg>
                    </button>
                    <button
                      class="text-mute hover:text-ink transition-colors cursor-pointer disabled:opacity-20 disabled:cursor-not-allowed"
                      :disabled="idx === agents.length - 1"
                      aria-label="Move down"
                      title="Move down"
                      @click="moveDown(idx)"
                    >
                      <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M6 9l6 6 6-6"/></svg>
                    </button>
                  </div>
                </div>

                <!-- Expanded body -->
                <div
                  v-if="expanded[agent.id]"
                  class="border-t border-hairline px-3 py-3 space-y-4"
                >
                  <!-- Display name -->
                  <div>
                    <label class="block text-xs text-mute mb-1.5">Display name</label>
                    <input
                      v-model="agent.name"
                      type="text"
                      :class="fieldClass + ' w-full'"
                      placeholder="Display name"
                      @change="persist"
                    >
                  </div>

                  <!-- Connection token -->
                  <div>
                    <div class="flex items-center justify-between mb-1.5">
                      <label class="text-xs text-mute">Connection token</label>
                      <div class="flex items-center gap-3">
                        <button
                          class="text-xs text-mute hover:text-ink transition-colors cursor-pointer"
                          @click="showTokens[agent.id] = !showTokens[agent.id]"
                        >{{ showTokens[agent.id] ? 'Hide' : 'Show' }}</button>
                        <button
                          class="text-xs text-mute hover:text-ink transition-colors cursor-pointer"
                          @click="generateTokenForAgent(agent)"
                        >Regenerate</button>
                      </div>
                    </div>
                    <input
                      v-model="agent.token"
                      :type="showTokens[agent.id] ? 'text' : 'password'"
                      class="w-full bg-canvas border border-hairline rounded px-2 py-1.5 text-xs font-mono text-body focus:outline-none focus:border-ink transition-colors"
                      @change="persist"
                    >
                  </div>

                  <!-- Footer actions -->
                  <div class="flex items-center justify-between pt-1">
                    <button
                      class="inline-flex items-center gap-1.5 px-3 py-1.5 border rounded text-xs font-medium transition-colors cursor-pointer"
                      :class="provisioning === agent.id
                        ? 'border-ink text-ink bg-surface-card'
                        : 'border-hairline text-body hover:text-ink hover:border-ink'"
                      @click="provisionAgent(agent)"
                    >
                      <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
                      {{ provisioning === agent.id ? 'Hide deploy' : 'Deploy to host' }}
                    </button>
                    <button
                      class="inline-flex items-center gap-1.5 text-xs text-mute hover:text-danger transition-colors cursor-pointer"
                      @click="askRemove(agent)"
                    >
                      <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18M8 6V4a1 1 0 011-1h6a1 1 0 011 1v2m2 0v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6"/></svg>
                      Remove
                    </button>
                  </div>

                  <!-- Deploy panel -->
                  <div
                    v-if="provisioning === agent.id"
                    class="border-t border-hairline -mx-3 px-3 pt-3 space-y-4"
                  >
                    <p class="text-xs text-mute">Run one of these on the target host to start the agent.</p>

                    <!-- Docker run -->
                    <div>
                      <div class="flex items-center justify-between mb-1.5">
                        <span class="text-xs font-medium text-body">Docker run</span>
                        <button :class="copyLink" @click="copy(provisionData.install, 'install')">{{ copied === 'install' ? '✓ Copied' : 'Copy' }}</button>
                      </div>
                      <pre :class="codeBlock"><code>{{ provisionData.install }}</code></pre>
                    </div>

                    <!-- Docker Compose -->
                    <div>
                      <div class="flex items-center justify-between mb-1.5">
                        <span class="text-xs font-medium text-body">docker-compose.yml</span>
                        <button :class="copyLink" @click="copy(provisionData.compose, 'compose')">{{ copied === 'compose' ? '✓ Copied' : 'Copy' }}</button>
                      </div>
                      <pre :class="codeBlock"><code>{{ provisionData.compose }}</code></pre>
                    </div>

                    <!-- Remove -->
                    <div>
                      <div class="flex items-center justify-between mb-1.5">
                        <span class="text-xs font-medium text-mute">Remove</span>
                        <button :class="copyLink" @click="copy(provisionData.remove, 'remove')">{{ copied === 'remove' ? '✓ Copied' : 'Copy' }}</button>
                      </div>
                      <pre :class="codeBlockMuted"><code>{{ provisionData.remove }}</code></pre>
                    </div>
                  </div>
                </div>
              </li>
            </ul>

            <!-- Auto-save status: every change is saved immediately. -->
            <div class="flex items-center gap-1.5 border-t border-hairline pt-3 pb-1 text-xs min-h-5">
              <template v-if="message">
                <svg
                  v-if="messageType === 'success'"
                  class="w-3.5 h-3.5 text-success shrink-0"
                  viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"
                ><polyline points="20 6 9 17 4 12"/></svg>
                <svg
                  v-else
                  class="w-3.5 h-3.5 text-danger shrink-0"
                  viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"
                ><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
                <span :class="messageType === 'error' ? 'text-danger' : 'text-mute'">{{ message }}</span>
              </template>
              <span
                v-else
                class="text-ash"
              >Changes are saved automatically.</span>
            </div>
          </div>
        </template>

        <template #password>
          <div class="max-w-sm">
            <p class="text-sm text-mute mb-4">
              Update the dashboard login. Change the username, the password, or both — your session stays active afterwards.
            </p>
            <div class="space-y-3">
              <div>
                <label class="block text-xs text-mute mb-1.5">Username</label>
                <input
                  v-model="username"
                  type="text"
                  autocomplete="username"
                  :class="fieldClass + ' w-full'"
                  placeholder="Username"
                  @keydown.enter="updateCredentials"
                >
              </div>
              <div>
                <label class="block text-xs text-mute mb-1.5">New password</label>
                <input
                  v-model="newPassword"
                  type="password"
                  autocomplete="new-password"
                  :class="fieldClass + ' w-full'"
                  placeholder="Leave blank to keep current"
                  @keydown.enter="updateCredentials"
                >
              </div>
              <div>
                <label class="block text-xs text-mute mb-1.5">Confirm new password</label>
                <input
                  v-model="confirmPassword"
                  type="password"
                  autocomplete="new-password"
                  :class="fieldClass + ' w-full'"
                  placeholder="Confirm new password"
                  @keydown.enter="updateCredentials"
                >
              </div>
              <div class="flex items-center gap-3 pt-1">
                <button
                  :class="btnPrimary"
                  :disabled="changingPassword"
                  @click="updateCredentials"
                >
                  {{ changingPassword ? 'Updating...' : 'Update login' }}
                </button>
                <span
                  v-if="passwordMessage"
                  class="text-xs"
                  :class="passwordMessageType === 'error' ? 'text-danger' : 'text-success'"
                >{{ passwordMessage }}</span>
              </div>
            </div>
          </div>
        </template>
      </Tabs>
    </div>

    <!-- Remove confirmation -->
    <Teleport to="body">
      <div
        v-if="pendingRemove"
        class="fixed inset-0 z-50 flex items-center justify-center bg-ink/40 p-4"
        role="dialog"
        aria-modal="true"
        aria-labelledby="remove-title"
        @click.self="cancelRemove"
      >
        <div class="w-full max-w-sm bg-canvas border border-hairline rounded-lg shadow-xl p-5">
          <div class="flex items-start gap-3">
            <div class="mt-0.5 w-8 h-8 rounded-full bg-surface-card flex items-center justify-center shrink-0">
              <svg class="w-4 h-4 text-danger" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18M8 6V4a1 1 0 011-1h6a1 1 0 011 1v2m2 0v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6"/></svg>
            </div>
            <div class="min-w-0">
              <h3
                id="remove-title"
                class="text-sm font-bold text-ink"
              >Remove agent?</h3>
              <p class="mt-1.5 text-xs text-mute leading-relaxed">
                <span class="font-medium text-body">{{ pendingRemove.name || pendingRemove.id }}</span>
                will be removed and its connection token revoked. This can't be undone.
              </p>
            </div>
          </div>
          <div class="mt-5 flex items-center justify-end gap-2">
            <button
              class="px-3 py-1.5 text-xs text-mute hover:text-ink transition-colors cursor-pointer"
              @click="cancelRemove"
            >Cancel</button>
            <button
              class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-danger hover:opacity-90 text-on-dark text-xs font-medium rounded transition-opacity cursor-pointer"
              @click="confirmRemove"
            >
              <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18M8 6V4a1 1 0 011-1h6a1 1 0 011 1v2m2 0v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6"/></svg>
              Remove
            </button>
          </div>
        </div>
      </div>
    </Teleport>
    </div>

    <!-- Sponsor and Coffee links -->
    <div class="flex items-center justify-center gap-4 text-xs font-mono text-mute">
      <a
        href="https://github.com/sponsors/kiwiprojekt"
        target="_blank"
        rel="noopener"
        class="inline-flex items-center gap-1.5 hover:text-ink transition-colors"
      >
        <svg aria-hidden="true" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M19 14c1.49-1.46 3-3.21 3-5.5A5.5 5.5 0 0 0 16.5 3c-1.76 0-3 .5-4.5 2-1.5-1.5-2.74-2-4.5-2A5.5 5.5 0 0 0 2 8.5c0 2.3 1.5 4.05 3 5.5l7 7Z"/></svg>
        Sponsor
      </a>
      <span class="text-ash/50 select-none">·</span>
      <a
        href="https://buycoffee.to/kiwiprojekt"
        target="_blank"
        rel="noopener"
        class="inline-flex items-center gap-1.5 hover:text-ink transition-colors"
      >
        <svg aria-hidden="true" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M17 8h1a4 4 0 1 1 0 8h-1"/><path d="M3 8h14v9a4 4 0 0 1-4 4H7a4 4 0 0 1-4-4Z"/><line x1="6" y1="2" x2="6" y2="4"/><line x1="10" y1="2" x2="10" y2="4"/><line x1="14" y1="2" x2="14" y2="4"/></svg>
        Buy me a coffee
      </a>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { generateToken, generateId, getCredentials, setCredentials } from '../utils.js'
import { useAdminApi } from '../composables/useAdminApi.js'
import Tabs from './Tabs.vue'

const { fetchAgents, saveAgents, fetchProvision, changeCredentials } = useAdminApi()

// Shared class tokens (kept in sync across the soft inputs / primary buttons)
const fieldClass = 'bg-surface-soft border border-hairline rounded px-2.5 py-1.5 text-ink text-xs placeholder-ash focus:outline-none focus:border-ink transition-colors'
const btnPrimary = 'px-4 py-2 bg-ink hover:bg-ink-deep text-on-dark text-sm font-medium rounded transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed'
const copyLink = 'text-xs text-mute hover:text-ink transition-colors cursor-pointer'
const codeBlock = 'bg-surface-dark text-on-dark text-xs font-mono leading-relaxed rounded p-3 overflow-x-auto whitespace-pre'
const codeBlockMuted = 'bg-surface-card text-body text-xs font-mono leading-relaxed rounded p-3 overflow-x-auto whitespace-pre'

const tabs = [
  { id: 'agents', label: 'Agents' },
  { id: 'password', label: 'Login' },
]

const username = ref(getCredentials().user)
const newPassword = ref('')
const confirmPassword = ref('')
const changingPassword = ref(false)
const passwordMessage = ref('')
const passwordMessageType = ref('success')

const agents = ref([])
const newName = ref('')
const message = ref('')
const messageType = ref('success')
const showTokens = reactive({})
const expanded = reactive({})
const provisioning = ref(null)
const provisionData = ref({ install: '', compose: '', remove: '' })
const copied = ref('')
const pendingRemove = ref(null)

onMounted(async () => {
  try {
    agents.value = await fetchAgents()
  } catch (e) {
    console.error('Failed to load agents:', e)
  }
})

let savedTimer = null
function flashSaved() {
  messageType.value = 'success'
  message.value = 'Saved'
  clearTimeout(savedTimer)
  savedTimer = setTimeout(() => {
    if (messageType.value === 'success') message.value = ''
  }, 1500)
}

// Every edit persists immediately (there is no Save button). On failure we
// refetch from the server so the optimistic UI never diverges from what was
// actually stored — otherwise a later Deploy could run against unsaved state.
async function persist() {
  try {
    await saveAgents(agents.value)
    flashSaved()
  } catch (e) {
    messageType.value = 'error'
    message.value = e.message || 'Save failed — reverted to saved state'
    try {
      agents.value = await fetchAgents()
    } catch (err) {
      console.error('Failed to resync agents:', err)
    }
  }
}

function toggle(id) {
  expanded[id] = !expanded[id]
}

function addAgent() {
  const name = newName.value.trim()
  if (!name) return
  // The ID is an opaque, auto-generated handle; the agent never types or sends
  // it (its token is its identity), so a UUID is fine and avoids collisions.
  const id = generateId()
  agents.value.push({ id, name, token: generateToken(), order: agents.value.length })
  expanded[id] = true
  newName.value = ''
  persist()
}

function generateTokenForAgent(agent) {
  agent.token = generateToken()
  persist()
}

function askRemove(agent) {
  pendingRemove.value = agent
}

function cancelRemove() {
  pendingRemove.value = null
}

function confirmRemove() {
  const agent = pendingRemove.value
  pendingRemove.value = null
  if (!agent) return
  const idx = agents.value.findIndex(a => a.id === agent.id)
  if (idx === -1) return
  if (provisioning.value === agent.id) provisioning.value = null
  agents.value.splice(idx, 1)
  persist()
}

function moveUp(idx) {
  if (idx === 0) return
  const tmp = agents.value[idx]; agents.value[idx] = agents.value[idx - 1]; agents.value[idx - 1] = tmp
  persist()
}

function moveDown(idx) {
  if (idx >= agents.value.length - 1) return
  const tmp = agents.value[idx]; agents.value[idx] = agents.value[idx + 1]; agents.value[idx + 1] = tmp
  persist()
}

function onKeydown(e) {
  if (e.key === 'Escape' && pendingRemove.value) cancelRemove()
}
onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))

async function provisionAgent(agent) {
  if (provisioning.value === agent.id) { provisioning.value = null; return }
  try {
    provisionData.value = await fetchProvision(agent.id)
    provisioning.value = agent.id; copied.value = ''
  } catch (e) { console.error('Failed to load provision data:', e) }
}

async function copy(text, key) {
  try {
    await navigator.clipboard.writeText(text)
    copied.value = key
    setTimeout(() => { if (copied.value === key) copied.value = '' }, 1800)
  } catch { /* clipboard unavailable */ }
}

async function updateCredentials() {
  passwordMessage.value = ''
  const current = getCredentials()
  const nextUser = username.value.trim()
  const wantsPassword = !!newPassword.value
  const wantsUsername = !!nextUser && nextUser !== current.user

  if (!wantsPassword && !wantsUsername) {
    passwordMessage.value = 'Nothing to change'
    passwordMessageType.value = 'error'
    return
  }
  if (wantsPassword && newPassword.value !== confirmPassword.value) {
    passwordMessage.value = 'Passwords do not match'
    passwordMessageType.value = 'error'
    return
  }

  changingPassword.value = true
  try {
    await changeCredentials(wantsUsername ? nextUser : '', wantsPassword ? newPassword.value : '')
    setCredentials(wantsUsername ? nextUser : current.user, wantsPassword ? newPassword.value : current.pass)
    newPassword.value = ''
    confirmPassword.value = ''
    passwordMessage.value = 'Login updated'
    passwordMessageType.value = 'success'
  } catch (e) {
    passwordMessage.value = e.message || 'Connection error'
    passwordMessageType.value = 'error'
  } finally { changingPassword.value = false }
}
</script>
