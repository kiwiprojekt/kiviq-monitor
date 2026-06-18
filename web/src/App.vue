<template>
  <div class="min-h-screen bg-canvas">
    <AppLogin
      v-if="!authenticated"
      @login="onLogin"
    />
    <AppLayout v-else />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import AppLogin from './components/AppLogin.vue'
import AppLayout from './components/AppLayout.vue'

const authenticated = ref(false)

import { getCredentials, setCredentials } from './utils.js'

onMounted(() => {
  const { user, pass } = getCredentials()
  if (user && pass) {
    authenticated.value = true
  }
})

function onLogin({ user, pass }) {
  setCredentials(user, pass)
  authenticated.value = true
}
</script>
