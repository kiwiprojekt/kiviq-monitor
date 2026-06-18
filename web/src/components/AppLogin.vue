<template>
  <div class="min-h-screen bg-canvas flex items-center justify-center p-4">
    <div class="w-full max-w-sm">
      <div class="text-center mb-8">
        <div class="w-12 h-12 rounded bg-charcoal flex items-center justify-center mx-auto mb-4">
          <span class="text-kiwi-sage text-lg font-bold font-mono">[+]</span>
        </div>
        <h1 class="text-2xl font-bold text-ink">
          kiviq
        </h1>
        <p class="text-sm text-mute mt-1">
          Server monitoring dashboard
        </p>
      </div>

      <form
        class="bg-canvas rounded-none border border-hairline p-6 space-y-4"
        @submit.prevent="handleLogin"
      >
        <div>
          <label
            for="login-user"
            class="block text-sm font-medium text-mute mb-1.5"
          >Username</label>
          <input
            id="login-user"
            v-model="user"
            type="text"
            autocomplete="username"
            class="w-full bg-surface-soft border border-hairline rounded px-3 py-2.5 text-ink text-sm placeholder-ash focus:outline-none focus:border-ink transition-colors"
            placeholder="Enter username"
            autofocus
          >
        </div>
        <div>
          <label
            for="login-pass"
            class="block text-sm font-medium text-mute mb-1.5"
          >Password</label>
          <div class="relative">
            <input
              id="login-pass"
              v-model="pass"
              :type="showPass ? 'text' : 'password'"
              autocomplete="current-password"
              class="w-full bg-surface-soft border border-hairline rounded px-3 py-2.5 pr-10 text-ink text-sm placeholder-ash focus:outline-none focus:border-ink transition-colors"
              placeholder="Enter password"
            >
            <button
              type="button"
              class="absolute right-2.5 top-1/2 -translate-y-1/2 text-mute hover:text-ink transition-colors p-1 cursor-pointer"
              :aria-label="showPass ? 'Hide password' : 'Show password'"
              @click="showPass = !showPass"
            >
              <svg
                v-if="!showPass"
                class="w-4 h-4"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" /><circle
                  cx="12"
                  cy="12"
                  r="3"
                />
              </svg>
              <svg
                v-else
                class="w-4 h-4"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
                <line
                  x1="1"
                  y1="1"
                  x2="23"
                  y2="23"
                />
              </svg>
            </button>
          </div>
        </div>
        <button
          type="submit"
          :disabled="loading"
          class="w-full bg-ink hover:bg-ink-deep disabled:opacity-50 disabled:cursor-not-allowed text-on-dark font-medium py-2.5 rounded transition-colors cursor-pointer"
        >
          <span
            v-if="loading"
            class="inline-flex items-center gap-2"
          >
            <svg
              class="animate-spin w-4 h-4"
              viewBox="0 0 24 24"
              fill="none"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              />
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
              />
            </svg>
            Signing in...
          </span>
          <span v-else>Login</span>
        </button>
        <p
          v-if="error"
          class="text-danger text-sm text-center"
          role="alert"
        >
          {{ error }}
        </p>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { apiUrl } from '../utils.js'

const emit = defineEmits(['login'])
const user = ref('')
const pass = ref('')
const error = ref('')
const loading = ref(false)
const showPass = ref(false)

async function handleLogin() {
  error.value = ''
  loading.value = true
  const auth = btoa(`${user.value}:${pass.value}`)
  try {
    const res = await fetch(apiUrl('/api/v1/agents'), {
      headers: { 'Authorization': `Basic ${auth}` },
    })
    if (res.ok) {
      emit('login', { user: user.value, pass: pass.value })
    } else {
      error.value = 'Invalid credentials'
    }
  } catch {
    error.value = 'Connection failed'
  } finally {
    loading.value = false
  }
}
</script>
