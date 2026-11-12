import { useAuthStore } from '../stores/auth'

const baseUrl = (import.meta.env.VITE_API_BASE_URL || '/api').replace(/\/$/, '')

export class ApiError extends Error {
  constructor(message, status) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

export async function request(path, options = {}) {
  const auth = useAuthStore()
  const headers = new Headers(options.headers || {})
  if (auth.token) headers.set('Authorization', `Bearer ${auth.token}`)
  if (options.body && !(options.body instanceof FormData)) headers.set('Content-Type', 'application/json')

  let response
  try {
    response = await fetch(`${baseUrl}${path}`, { ...options, headers })
  } catch {
    throw new ApiError('Unable to connect. Check your connection and try again.', 0)
  }

  if (response.status === 401) {
    auth.logout()
    if (window.location.pathname !== '/login' && window.location.pathname !== '/register') {
      window.location.assign('/login')
    }
  }

  const contentType = response.headers.get('content-type') || ''
  const payload = contentType.includes('application/json')
    ? await response.json().catch(() => null)
    : null
  if (!response.ok) {
    throw new ApiError(payload?.error || `Request failed (${response.status}).`, response.status)
  }
  return payload
}

export const api = {
  login: (credentials) => request('/auth/login', { method: 'POST', body: JSON.stringify(credentials) }),
  register: (details) => request('/auth/register', { method: 'POST', body: JSON.stringify(details) }),
  me: () => request('/me'),
  listTickets: (params) => {
    const query = new URLSearchParams()
    Object.entries(params).forEach(([key, value]) => {
      if (value !== '' && value !== null && value !== undefined) query.set(key, value)
    })
    return request(`/tickets?${query}`)
  },
  createTicket: (ticket) => request('/tickets', { method: 'POST', body: JSON.stringify(ticket) }),
  ticket: (id) => request(`/tickets/${encodeURIComponent(id)}`),
  updateTicket: (id, changes) => request(`/tickets/${encodeURIComponent(id)}`, { method: 'PATCH', body: JSON.stringify(changes) }),
  deleteTicket: (id) => request(`/tickets/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  addComment: (id, comment) => request(`/tickets/${encodeURIComponent(id)}/comments`, { method: 'POST', body: JSON.stringify(comment) }),
  users: () => request('/users'),
  agents: () => request('/agents'),
  updateUserRole: (id, role) => request(`/users/${encodeURIComponent(id)}/role`, { method: 'PATCH', body: JSON.stringify({ role }) }),
}
