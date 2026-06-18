<template>
  <div class="space-y-5">
    <SystemInfo :agent="agent" />
    <hr class="border-hairline">
    <CpuSection :agent="agent" />
    <hr class="border-hairline">
    <MemorySection :agent="agent" />
    <hr class="border-hairline">
    <DiskTable :agent="agent" />
    <hr class="border-hairline">
    <NetworkSection :agent="agent" />
    <template v-if="agent.gpu?.name">
      <hr class="border-hairline">
      <GpuSection :agent="agent" />
    </template>
  </div>
</template>

<script setup>
import SystemInfo from './SystemInfo.vue'
import CpuSection from './CpuSection.vue'
import MemorySection from './MemorySection.vue'
import DiskTable from './DiskTable.vue'
import NetworkSection from './NetworkSection.vue'
import GpuSection from './GpuSection.vue'

defineProps({ agent: Object })
</script>
