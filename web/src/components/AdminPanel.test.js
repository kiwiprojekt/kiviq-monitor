import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

const { fetchAgentsMock, saveAgentsMock, fetchProvisionMock, changeCredentialsMock } = vi.hoisted(() => ({
  fetchAgentsMock: vi.fn(),
  saveAgentsMock: vi.fn(),
  fetchProvisionMock: vi.fn(),
  changeCredentialsMock: vi.fn(),
}))

vi.mock('../composables/useAdminApi.js', () => ({
  useAdminApi: () => ({
    fetchAgents: fetchAgentsMock,
    saveAgents: saveAgentsMock,
    fetchProvision: fetchProvisionMock,
    changeCredentials: changeCredentialsMock,
  }),
}))

import AdminPanel from './AdminPanel.vue'

const mountOpts = { global: { stubs: { teleport: true } } }

function byText(wrapper, selector, text) {
  return wrapper.findAll(selector).find(el => el.text().includes(text))
}

beforeEach(() => {
  vi.stubGlobal('localStorage', { getItem: () => null, setItem: () => {} })
  fetchAgentsMock.mockReset().mockResolvedValue([])
  saveAgentsMock.mockReset().mockResolvedValue(null)
  fetchProvisionMock.mockReset()
  changeCredentialsMock.mockReset()
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('AdminPanel auto-save', () => {
  it('persists immediately when an agent is added', async () => {
    const wrapper = mount(AdminPanel, mountOpts)
    await flushPromises()

    await wrapper.find('input[placeholder="Display name"]').setValue('web-1')
    await byText(wrapper, 'button', 'Add agent').trigger('click')
    await flushPromises()

    expect(saveAgentsMock).toHaveBeenCalledTimes(1)
    const saved = saveAgentsMock.mock.calls[0][0]
    expect(saved.some(a => a.name === 'web-1')).toBe(true)
  })

  it('refetches from the server when a save fails, so the UI cannot diverge', async () => {
    const wrapper = mount(AdminPanel, mountOpts)
    await flushPromises()
    fetchAgentsMock.mockClear()
    saveAgentsMock.mockRejectedValueOnce(new Error('boom'))

    await wrapper.find('input[placeholder="Display name"]').setValue('web-2')
    await byText(wrapper, 'button', 'Add agent').trigger('click')
    await flushPromises()

    expect(saveAgentsMock).toHaveBeenCalled()
    expect(fetchAgentsMock).toHaveBeenCalledTimes(1) // resync after failure
    expect(wrapper.text()).toContain('boom')
  })
})

describe('AdminPanel remove confirmation', () => {
  async function mountWithOneAgent() {
    fetchAgentsMock.mockResolvedValue([{ id: 'a', name: 'Alpha', token: 't', order: 0 }])
    const wrapper = mount(AdminPanel, mountOpts)
    await flushPromises()
    // Expand the agent so its Remove button is visible.
    await wrapper.find('button[aria-expanded]').trigger('click')
    return wrapper
  }

  it('does not remove until the dialog is confirmed', async () => {
    const wrapper = await mountWithOneAgent()

    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
    await byText(wrapper, 'button', 'Remove').trigger('click')

    const dialog = wrapper.find('[role="dialog"]')
    expect(dialog.exists()).toBe(true)
    expect(dialog.text()).toContain('Alpha')
    expect(saveAgentsMock).not.toHaveBeenCalled() // opening the dialog saves nothing
  })

  it('cancelling closes the dialog and keeps the agent', async () => {
    const wrapper = await mountWithOneAgent()
    await byText(wrapper, 'button', 'Remove').trigger('click')

    await byText(wrapper, 'button', 'Cancel').trigger('click')
    await flushPromises()

    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
    expect(saveAgentsMock).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Alpha')
  })

  it('confirming removes the agent and persists', async () => {
    const wrapper = await mountWithOneAgent()
    await byText(wrapper, 'button', 'Remove').trigger('click')

    // The dialog's own Remove button confirms.
    const confirm = wrapper.find('[role="dialog"]').findAll('button').find(b => b.text().includes('Remove'))
    await confirm.trigger('click')
    await flushPromises()

    expect(saveAgentsMock).toHaveBeenCalledTimes(1)
    expect(saveAgentsMock.mock.calls[0][0]).toEqual([])
  })
})
