import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import StatBar from './StatBar.vue'

describe('StatBar', () => {
  it('renders label', () => {
    const wrapper = mount(StatBar, { props: { label: 'CPU', value: 50 } })
    expect(wrapper.text()).toContain('CPU')
  })

  it('displays value as percentage', () => {
    const wrapper = mount(StatBar, { props: { label: 'RAM', value: 73.4 } })
    expect(wrapper.text()).toContain('73.4%')
  })

  it('caps value at 100', () => {
    const wrapper = mount(StatBar, { props: { label: 'CPU', value: 150 } })
    expect(wrapper.text()).toContain('100.0%')
    const bar = wrapper.find('[style]')
    expect(bar.attributes('style')).toContain('width: 100%')
  })

  it('handles null value', () => {
    const wrapper = mount(StatBar, { props: { label: 'CPU', value: null } })
    expect(wrapper.text()).toContain('0.0%')
  })

  it('handles NaN value', () => {
    const wrapper = mount(StatBar, { props: { label: 'CPU', value: NaN } })
    expect(wrapper.text()).toContain('0.0%')
  })

  it('applies danger class for high values', () => {
    const wrapper = mount(StatBar, { props: { label: 'CPU', value: 95 } })
    expect(wrapper.find('.text-danger').exists()).toBe(true)
    expect(wrapper.find('.bg-danger').exists()).toBe(true)
  })

  it('applies warning class for medium values', () => {
    const wrapper = mount(StatBar, { props: { label: 'CPU', value: 80 } })
    expect(wrapper.find('.text-warning').exists()).toBe(true)
    expect(wrapper.find('.bg-warning').exists()).toBe(true)
  })

  it('applies success class for low values', () => {
    const wrapper = mount(StatBar, { props: { label: 'CPU', value: 50 } })
    expect(wrapper.find('.text-success').exists()).toBe(true)
    expect(wrapper.find('.bg-success').exists()).toBe(true)
  })

  it('sets bar width to match value', () => {
    const wrapper = mount(StatBar, { props: { label: 'CPU', value: 42.7 } })
    const bar = wrapper.findAll('div').find(el => el.classes().includes('bg-success') || el.classes().includes('bg-warning') || el.classes().includes('bg-danger'))
    expect(bar.attributes('style')).toContain('width: 42.7%')
  })

  it('includes tooltip with label and value', () => {
    const wrapper = mount(StatBar, { props: { label: 'Disk', value: 65.3 } })
    expect(wrapper.attributes('title')).toBe('Disk: 65.3%')
  })

  it('renders trailing slot content after the value', () => {
    const wrapper = mount(StatBar, {
      props: { label: 'CPU', value: 50 },
      slots: { trailing: '<span class="freq">2400</span>' },
    })
    expect(wrapper.find('.freq').exists()).toBe(true)
    expect(wrapper.find('.freq').text()).toBe('2400')
  })

  it('colors the value bar from the threshold but leaves the value neutral when coloredValue is false', () => {
    const wrapper = mount(StatBar, { props: { label: 'core', value: 95, coloredValue: false } })
    // The bar still reflects the danger threshold...
    expect(wrapper.find('.bg-danger').exists()).toBe(true)
    // ...but the numeric value is rendered neutrally, not in a threshold color.
    expect(wrapper.find('.text-danger').exists()).toBe(false)
    expect(wrapper.find('.text-warning').exists()).toBe(false)
    expect(wrapper.find('.text-success').exists()).toBe(false)
  })
})
