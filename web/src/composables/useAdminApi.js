import { getAuth, apiUrl } from '../utils.js'

// One contract for every admin call: throw an Error (carrying the server's
// message) on a non-ok response, return the parsed JSON body (or null when
// empty) on success. Call sites use a single try/catch shape.
async function request(url, { method = 'GET', body } = {}) {
  const headers = { 'Authorization': `Basic ${getAuth()}` }
  if (body !== undefined) headers['Content-Type'] = 'application/json'

  const res = await fetch(apiUrl(url), { method, headers, body })
  if (!res.ok) {
    throw new Error((await res.text()) || `Request failed (${res.status})`)
  }
  const text = await res.text()
  return text ? JSON.parse(text) : null
}

export function useAdminApi() {
  const fetchAgents = () => request('/api/v1/admin/agents')

  const saveAgents = (agents) =>
    request('/api/v1/admin/agents', {
      method: 'PUT',
      body: JSON.stringify(
        agents.map((s, i) => ({
          id: s.id,
          name: s.name || s.id,
          token: s.token || '',
          order: i,
        })),
      ),
    })

  const fetchProvision = (agentId) =>
    request(`/api/v1/admin/provision/${agentId}`)

  const changeCredentials = (newUsername, newPassword) =>
    request('/api/v1/admin/password', {
      method: 'POST',
      body: JSON.stringify({ new_username: newUsername, new_password: newPassword }),
    })

  return { fetchAgents, saveAgents, fetchProvision, changeCredentials }
}
