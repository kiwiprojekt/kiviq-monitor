<template>
  <section>
    <h3 class="text-xs font-bold text-ink mb-2">
      Disks
    </h3>
    <div class="overflow-x-auto -mx-4 px-4">
      <table class="w-full text-xs">
        <thead>
          <tr class="text-mute text-left">
            <th class="pb-1.5 font-medium">
              Device
            </th>
            <th class="pb-1.5 font-medium">
              Mount
            </th>
            <th class="pb-1.5 font-medium">
              Type
            </th>
            <th class="pb-1.5 font-medium text-right">
              Total
            </th>
            <th class="pb-1.5 font-medium text-right">
              Used
            </th>
            <th class="pb-1.5 font-medium text-right">
              Free
            </th>
            <th class="pb-1.5 font-medium text-right">
              Usage
            </th>
            <th class="pb-1.5 font-medium text-right">
              Read
            </th>
            <th class="pb-1.5 font-medium text-right">
              Write
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="d in (agent.disk || [])"
            :key="d.device"
            class="border-t border-hairline"
          >
            <td class="py-1.5 text-ink font-mono">
              {{ d.device }}
            </td>
            <td class="py-1.5 text-body">
              {{ d.mount }}
            </td>
            <td class="py-1.5 text-stone">
              {{ d.fstype }}
            </td>
            <td class="py-1.5 text-body text-right font-mono tabular-nums">
              {{ formatBytes(d.total_bytes) }}
            </td>
            <td class="py-1.5 text-body text-right font-mono tabular-nums">
              {{ formatBytes(d.used_bytes) }}
            </td>
            <td class="py-1.5 text-body text-right font-mono tabular-nums">
              {{ formatBytes(d.free_bytes) }}
            </td>
            <td
              class="py-1.5 text-right font-mono tabular-nums"
              :class="textColor(d.usage_percent)"
            >
              {{ d.usage_percent.toFixed(1) }}%
            </td>
            <td class="py-1.5 text-body text-right font-mono tabular-nums">
              {{ formatBytes(d.read_bytes) }}
            </td>
            <td class="py-1.5 text-body text-right font-mono tabular-nums">
              {{ formatBytes(d.write_bytes) }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<script setup>
import { formatBytes, textColor } from '../utils.js'

defineProps({ agent: Object })
</script>
