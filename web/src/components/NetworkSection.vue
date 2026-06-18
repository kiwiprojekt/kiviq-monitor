<template>
  <section>
    <h3 class="text-xs font-bold text-ink mb-2">
      Network
    </h3>
    <div class="overflow-x-auto -mx-4 px-4">
      <table class="w-full text-xs">
        <thead>
          <tr class="text-mute text-left">
            <th class="pb-1.5 font-medium">
              Interface
            </th>
            <th class="pb-1.5 font-medium">
              IP
            </th>
            <th class="pb-1.5 font-medium">
              Speed
            </th>
            <th class="pb-1.5 font-medium text-right">
              RX
            </th>
            <th class="pb-1.5 font-medium text-right">
              TX
            </th>
            <th class="pb-1.5 font-medium text-right">
              Pkts In
            </th>
            <th class="pb-1.5 font-medium text-right">
              Pkts Out
            </th>
            <th class="pb-1.5 font-medium text-right">
              Errors
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="n in (agent.network || [])"
            :key="n.interface"
            class="border-t border-hairline"
          >
            <td class="py-1.5 text-ink font-medium">
              {{ n.interface }}
            </td>
            <td class="py-1.5 text-body font-mono">
              {{ n.ip || '—' }}
            </td>
            <td class="py-1.5 text-body">
              {{ n.speed_mbps > 0 ? n.speed_mbps + ' Mbps' : 'N/A' }}
            </td>
            <td class="py-1.5 text-body text-right font-mono tabular-nums">
              {{ formatBytes(n.bytes_in) }}
            </td>
            <td class="py-1.5 text-body text-right font-mono tabular-nums">
              {{ formatBytes(n.bytes_out) }}
            </td>
            <td class="py-1.5 text-body text-right font-mono tabular-nums">
              {{ n.packets_in.toLocaleString() }}
            </td>
            <td class="py-1.5 text-body text-right font-mono tabular-nums">
              {{ n.packets_out.toLocaleString() }}
            </td>
            <td
              class="py-1.5 text-right font-mono tabular-nums"
              :class="(n.errors_in + n.errors_out) > 0 ? 'text-danger' : 'text-body'"
            >
              {{ n.errors_in + n.errors_out }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div class="mt-3 grid grid-cols-1 md:grid-cols-2 gap-3">
      <StatChart
        :agent-id="agent.agent_id"
        label="RX"
        kpi-prefix="RX "
        field="rx"
        color="#10b981"
        unit=" B"
        :max-val="1"
        :compute-rate="true"
      />
      <StatChart
        :agent-id="agent.agent_id"
        label="TX"
        kpi-prefix="TX "
        field="tx"
        color="#f59e0b"
        unit=" B"
        :max-val="1"
        :compute-rate="true"
      />
    </div>
  </section>
</template>

<script setup>
import StatChart from './StatChart.vue'
import { formatBytes } from '../utils.js'

defineProps({
  agent: Object,
})
</script>
