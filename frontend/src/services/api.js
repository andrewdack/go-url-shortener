const apiBaseUrl = (import.meta.env.VITE_API_BASE_URL || 'http://localhost:9808').replace(/\/$/, '')

export class ApiError extends Error {
  constructor(message, status) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function request(path, options = {}) {
  let response

  try {
    response = await fetch(`${apiBaseUrl}${path}`, {
      ...options,
      headers: {
        ...(options.body ? { 'Content-Type': 'application/json' } : {}),
        ...options.headers,
      },
    })
  } catch {
    throw new ApiError('Could not reach the URL service. Make sure the backend is running.', 0)
  }

  const payload = await response.json().catch(() => ({}))

  if (!response.ok) {
    throw new ApiError(payload.error || 'Something went wrong. Please try again.', response.status)
  }

  return payload
}

export function createShortUrl(longUrl, userId) {
  return request('/create-short-url', {
    method: 'POST',
    body: JSON.stringify({ longUrl, userId }),
  })
}

export function getClickCount(shortCode) {
  return request(`/${encodeURIComponent(shortCode)}/count`)
}

export function updateShortUrl(shortCode, longUrl, userId) {
  return request(`/${encodeURIComponent(shortCode)}`, {
    method: 'PATCH',
    body: JSON.stringify({ longUrl, userId }),
  })
}

export function deleteShortUrl(shortCode, userId) {
  return request(`/${encodeURIComponent(shortCode)}`, {
    method: 'DELETE',
    body: JSON.stringify({ userId }),
  })
}
