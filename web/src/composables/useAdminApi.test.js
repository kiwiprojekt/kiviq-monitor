import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { useAdminApi } from './useAdminApi.js'

let fetchMock

beforeEach(() => {
  fetchMock = vi.fn()
  vi.stubGlobal('fetch', fetchMock)
  vi.stubGlobal('localStorage', { getItem: () => null, setItem: () => {} })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

function jsonResponse(body, ok = true, status = 200) {
  return { ok, status, text: async () => (body == null ? '' : JSON.stringify(body)) }
}

describe('useAdminApi unified contract', () => {
  it('returns parsed JSON on success', async () => {
    fetchMock.mockResolvedValue(jsonResponse([{ id: 'a' }]))
    const { fetchAgents } = useAdminApi()
    await expect(fetchAgents()).resolves.toEqual([{ id: 'a' }])
  })

  it('throws with the server message on a non-ok response', async () => {
    fetchMock.mockResolvedValue({ ok: false, status: 400, text: async () => 'new username must not be empty' })
    const { changeCredentials } = useAdminApi()
    await expect(changeCredentials('   ', 'pw')).rejects.toThrow('new username must not be empty')
  })

  it('returns null on an empty success body', async () => {
    fetchMock.mockResolvedValue(jsonResponse(null))
    const { saveAgents } = useAdminApi()
    await expect(saveAgents([])).resolves.toBeNull()
  })

  it('sends JSON content-type only when there is a body', async () => {
    fetchMock.mockResolvedValue(jsonResponse([]))
    const { fetchAgents, saveAgents } = useAdminApi()

    await fetchAgents()
    expect(fetchMock.mock.calls[0][1].headers['Content-Type']).toBeUndefined()

    await saveAgents([{ id: 'a', name: 'A', token: 't' }])
    expect(fetchMock.mock.calls[1][1].method).toBe('PUT')
    expect(fetchMock.mock.calls[1][1].headers['Content-Type']).toBe('application/json')
  })
})
