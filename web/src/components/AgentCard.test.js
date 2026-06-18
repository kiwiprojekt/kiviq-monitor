import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import AgentCard from './AgentCard.vue'

const makeAgent = (overrides = {}) => ({
  agent_id: 'srv-01',
  agent_name: 'Web Server',
  hostname: 'web-prod',
  // Online/offline is derived client-side from last_seen, not the snapshot flag.
  last_seen: new Date().toISOString(),
  uptime_seconds: 90061,
  cpu: { usage_percent: 45.2, model_name: 'Intel Xeon' },
  memory: { usage_percent: 62.8 },
  disk: [
    { device: '/dev/sda1', usage_percent: 73.5 },
    { device: '/dev/nvme0n1p1', usage_percent: 45.0 },
  ],
  network: [
    { interface: 'eth0', bytes_in: 1073741824, bytes_out: 536870912, speed_mbps: 1000 },
  ],
  docker: { containers: [{ id: 'abc' }, { id: 'def' }, { id: 'ghi' }] },
  ...overrides,
})

describe('AgentCard', () => {
  it('displays agent name', () => {
    const wrapper = mount(AgentCard, { props: { agent: makeAgent() } })
    expect(wrapper.text()).toContain('Web Server')
  })

  it('falls back to hostname when no agent_name', () => {
    const wrapper = mount(AgentCard, {
      props: { agent: makeAgent({ agent_name: '' }) },
    })
    expect(wrapper.text()).toContain('web-prod')
  })

  it('shows online indicator for a recent last_seen', () => {
    const wrapper = mount(AgentCard, {
      props: { agent: makeAgent({ last_seen: new Date().toISOString() }) },
    })
    expect(wrapper.find('.bg-success').exists()).toBe(true)
  })

  it('shows offline indicator for a stale last_seen', () => {
    const wrapper = mount(AgentCard, {
      props: { agent: makeAgent({ last_seen: new Date(Date.now() - 120000).toISOString() }) },
    })
    expect(wrapper.find('.bg-ash').exists()).toBe(true)
    expect(wrapper.text()).toContain('offline')
  })

  it('formats uptime correctly', () => {
    const wrapper = mount(AgentCard, {
      props: { agent: makeAgent({ uptime_seconds: 90061 }) },
    })
    expect(wrapper.text()).toContain('1d 1h')
  })

  it('shows CPU usage', () => {
    const wrapper = mount(AgentCard, {
      props: { agent: makeAgent() },
    })
    expect(wrapper.text()).toContain('45.2%')
  })

  it('shows RAM usage', () => {
    const wrapper = mount(AgentCard, {
      props: { agent: makeAgent() },
    })
    expect(wrapper.text()).toContain('62.8%')
  })

  it('shows CPU model name', () => {
    const wrapper = mount(AgentCard, {
      props: { agent: makeAgent() },
    })
    expect(wrapper.text()).toContain('Intel Xeon')
  })

  it('shows disk devices with usage', () => {
    const wrapper = mount(AgentCard, {
      props: { agent: makeAgent() },
    })
    expect(wrapper.text()).toContain('/dev/sda1')
    expect(wrapper.text()).toContain('74%')
    expect(wrapper.text()).toContain('/dev/nvme0n1p1')
  })

  it('shows network interface with formatted bytes', () => {
    const wrapper = mount(AgentCard, {
      props: { agent: makeAgent() },
    })
    expect(wrapper.text()).toContain('eth0')
    expect(wrapper.text()).toContain('1000M')
  })

  it('shows container count', () => {
    const wrapper = mount(AgentCard, {
      props: { agent: makeAgent() },
    })
    expect(wrapper.text()).toContain('3 containers')
  })

  it('hides container section when none exist', () => {
    const wrapper = mount(AgentCard, {
      props: { agent: makeAgent({ docker: {} }) },
    })
    expect(wrapper.text()).not.toContain('containers')
  })

  it('emits select event on click', async () => {
    const wrapper = mount(AgentCard, {
      props: { agent: makeAgent() },
    })
    await wrapper.trigger('keydown.enter')
    expect(wrapper.emitted('select')).toBeTruthy()
    expect(wrapper.emitted('select')[0][0].agent_id).toBe('srv-01')
  })

  it('hides disk section when no disks', () => {
    const wrapper = mount(AgentCard, {
      props: { agent: makeAgent({ disk: [] }) },
    })
    expect(wrapper.text()).not.toContain('Disks')
  })

  it('hides network section when no network', () => {
    const wrapper = mount(AgentCard, {
      props: { agent: makeAgent({ network: [] }) },
    })
    expect(wrapper.text()).not.toContain('Network')
  })
})
