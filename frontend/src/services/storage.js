const USER_ID_KEY = 'tiny-route:user-id'
const LINKS_KEY = 'tiny-route:links'

function newUserId() {
  if (typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }

  const values = crypto.getRandomValues(new Uint32Array(4))
  return Array.from(values, (value) => value.toString(16).padStart(8, '0')).join('-')
}

export function getUserId() {
  let userId = localStorage.getItem(USER_ID_KEY)

  if (!userId) {
    userId = newUserId()
    localStorage.setItem(USER_ID_KEY, userId)
  }

  return userId
}

export function getSavedLinks() {
  try {
    const links = JSON.parse(localStorage.getItem(LINKS_KEY) || '[]')
    return Array.isArray(links) ? links : []
  } catch {
    return []
  }
}

function persistLinks(links) {
  localStorage.setItem(LINKS_KEY, JSON.stringify(links))
}

export function shortCodeFromUrl(shortUrl) {
  const path = new URL(shortUrl).pathname.replace(/\/$/, '')
  return decodeURIComponent(path.split('/').pop())
}

export function rememberLink({ shortUrl, longUrl }) {
  const links = getSavedLinks()
  const shortCode = shortCodeFromUrl(shortUrl)
  const existing = links.find((link) => link.shortCode === shortCode)

  if (existing) {
    existing.longUrl = longUrl
    existing.shortUrl = shortUrl
  } else {
    links.unshift({
      shortCode,
      shortUrl,
      longUrl,
      createdAt: new Date().toISOString(),
    })
  }

  persistLinks(links)
}

export function updateSavedLink(shortCode, longUrl) {
  const links = getSavedLinks()
  const link = links.find((item) => item.shortCode === shortCode)

  if (link) {
    link.longUrl = longUrl
    persistLinks(links)
  }
}

export function forgetLink(shortCode) {
  persistLinks(getSavedLinks().filter((link) => link.shortCode !== shortCode))
}
