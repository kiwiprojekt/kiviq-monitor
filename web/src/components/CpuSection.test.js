import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import CpuSection from './CpuSection.vue'
import StatBar from './StatBar.vue'

const agent = {
  agent_id: 'a1',
  cpu: {
    usage_percent: 42.5,
    per_core: [10, 95],
    freq_mhz: [2400, 3200],
    model_name: 'Test CPU',
    freq_min_mhz: 800,
    freq_max_mhz: 4000,
    temperatures: [{ label: 'core', celsius: 55 }],
  },
}

function mountCpu() {
  return mount(CpuSection, { props: { agent }, global: { stubs: { StatChart: true } } })
}

describe('CpuSection', () => {
  // The overall bar and every per-core bar should reuse the canonical StatBar
  // rather than re-inlining bar markup and color logic.
  it('renders one StatBar for the overall usage plus one per core', () => {
    const bars = mountCpu().findAllComponents(StatBar)
    expect(bars.length).toBe(1 + agent.cpu.per_core.length)
  })

  it('colors the overall value but renders per-core values neutrally', () => {
    const bars = mountCpu().findAllComponents(StatBar)
    expect(bars[0].props('coloredValue')).not.toBe(false)
    expect(bars[1].props('coloredValue')).toBe(false)
    expect(bars[2].props('coloredValue')).toBe(false)
  })

  it('shows overall usage, per-core usage, and per-core frequencies', () => {
    const text = mountCpu().text()
    expect(text).toContain('42.5%')
    expect(text).toContain('10.0%')
    expect(text).toContain('95.0%')
    expect(text).toContain('2400')
    expect(text).toContain('3200')
  })

  it('renders the CPU model and temperature', () => {
    const text = mountCpu().text()
    expect(text).toContain('Test CPU')
    expect(text).toContain('55.0')
  })
})
