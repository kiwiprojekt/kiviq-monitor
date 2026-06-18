import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import DockerContainerRow from './DockerContainerRow.vue'

const mockContext = {
  scale: vi.fn(),
  clearRect: vi.fn(),
  beginPath: vi.fn(),
  moveTo: vi.fn(),
  lineTo: vi.fn(),
  closePath: vi.fn(),
  fill: vi.fn(),
  stroke: vi.fn(),
}

// Mock canvas and getBoundingClientRect so they don't throw or block in tests.
HTMLCanvasElement.prototype.getContext = vi.fn(() => mockContext)
HTMLCanvasElement.prototype.getBoundingClientRect = vi.fn(() => ({
  width: 100,
  height: 36,
}))

function mountContainerRow(props) {
  return mount(DockerContainerRow, {
    props,
  })
}

describe('DockerContainerRow', () => {
  const mockContainer = {
    id: 'abc123xyz789',
    name: 'test-app',
    image: 'nginx:latest',
    state: 'running',
    status: 'Up 5 minutes',
    cpu_percent: 0.0,
    memory_usage_bytes: 52428800, // 50MB
    memory_limit_bytes: 104857600, // 100MB
    network_rx_bytes: 1000,
    network_tx_bytes: 2000,
    ports: [
      { ip: '0.0.0.0', private_port: 80, public_port: 8080, type: 'tcp' },
      { ip: '127.0.0.1', private_port: 443, public_port: 8443, type: 'tcp' },
    ],
  }

  it('renders general container details when expanded', async () => {
    const wrapper = mountContainerRow({ container: mockContainer })
    expect(wrapper.text()).toContain('test-app')
    expect(wrapper.text()).not.toContain('Image') // not expanded yet

    await wrapper.trigger('click') // expand
    expect(wrapper.text()).toContain('Image')
    expect(wrapper.text()).toContain('nginx:latest')
    expect(wrapper.text()).toContain('abc123xyz789'.slice(0, 12))
    expect(wrapper.text()).toContain('50.0 MB')
  })

  it('renders bound ports as clickable links using agentIp fallback to localhost', async () => {
    const wrapper = mountContainerRow({
      container: mockContainer,
      agentIp: '192.168.1.100',
    })
    await wrapper.trigger('click') // expand

    const links = wrapper.findAll('a')
    expect(links.length).toBe(2)

    // First port is bound to 0.0.0.0, so it uses the agentIp prop
    expect(links[0].text()).toBe('8080->80/tcp')
    expect(links[0].attributes('href')).toBe('http://192.168.1.100:8080')
    expect(links[0].attributes('target')).toBe('_blank')

    // Second port is bound to 127.0.0.1 specifically, so it uses 127.0.0.1
    expect(links[1].text()).toBe('8443->443/tcp')
    expect(links[1].attributes('href')).toBe('http://127.0.0.1:8443')
  })

  it('renders "none" when there are no ports or no bound ports', async () => {
    const wrapper = mountContainerRow({
      container: { ...mockContainer, ports: [] },
    })
    await wrapper.trigger('click') // expand
    expect(wrapper.text()).toContain('Ports')
    expect(wrapper.text()).toContain('none')
  })

  it('prevents click propagation on port links to avoid collapsing the card', async () => {
    const wrapper = mountContainerRow({
      container: mockContainer,
      agentIp: '192.168.1.100',
    })
    await wrapper.trigger('click') // expand
    expect(wrapper.vm.expanded).toBe(true)

    const link = wrapper.find('a')
    await link.trigger('click') // click the link

    // The card should still be expanded because we called stopPropagation/click.stop
    expect(wrapper.vm.expanded).toBe(true)
  })

  it('accumulates a history point per container snapshot even when CPU stays constant', async () => {
    // The watcher keys on the whole container object, not cpu_percent, so a new
    // snapshot with identical CPU still pushes a point. drawSpark only strokes a
    // path once history has >= 2 points, so accumulation is observable on the
    // canvas: one point strokes nothing, a second one strokes the line.
    mockContext.stroke.mockClear()
    const wrapper = mountContainerRow({ container: mockContainer })
    await wrapper.trigger('click') // expand -> redraw with a single history point
    await nextTick()
    await nextTick()

    // One point so far (immediate watch); drawSpark returns before stroking.
    expect(mockContext.stroke).not.toHaveBeenCalled()

    // Second snapshot, same constant CPU -> second history point -> line strokes.
    await wrapper.setProps({ container: { ...mockContainer } })
    await nextTick()
    await nextTick()
    expect(mockContext.stroke).toHaveBeenCalled()
  })
})
