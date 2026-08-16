<script setup>
import { onMounted, reactive, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { ApiError, deleteShortUrl, getClickCount, updateShortUrl } from '../services/api'
import { forgetLink, getSavedLinks, getUserId, updateSavedLink } from '../services/storage'

const links = ref([])
const counts = reactive({})
const countErrors = reactive({})
const editingCode = ref('')
const editUrl = ref('')
const deletingCode = ref('')
const busyCode = ref('')
const actionError = ref('')
const copiedCode = ref('')

onMounted(() => {
  links.value = getSavedLinks()
  links.value.forEach(loadCount)
})

async function loadCount(link) {
  countErrors[link.shortCode] = ''
  try {
    const payload = await getClickCount(link.shortCode)
    counts[link.shortCode] = payload.clicks
  } catch (error) {
    countErrors[link.shortCode] = error instanceof ApiError && error.status === 404 ? 'Expired' : 'Unavailable'
  }
}

function startEditing(link) {
  actionError.value = ''
  deletingCode.value = ''
  editingCode.value = link.shortCode
  editUrl.value = link.longUrl
}

async function saveEdit(link) {
  actionError.value = ''
  busyCode.value = link.shortCode

  try {
    await updateShortUrl(link.shortCode, editUrl.value.trim(), getUserId())
    link.longUrl = editUrl.value.trim()
    updateSavedLink(link.shortCode, link.longUrl)
    editingCode.value = ''
  } catch (error) {
    actionError.value = friendlyActionError(error, 'update')
  } finally {
    busyCode.value = ''
  }
}

async function confirmDelete(link) {
  actionError.value = ''
  busyCode.value = link.shortCode

  try {
    await deleteShortUrl(link.shortCode, getUserId())
    forgetLink(link.shortCode)
    links.value = links.value.filter((item) => item.shortCode !== link.shortCode)
    deletingCode.value = ''
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) {
      forgetLink(link.shortCode)
      links.value = links.value.filter((item) => item.shortCode !== link.shortCode)
      deletingCode.value = ''
      return
    }
    actionError.value = friendlyActionError(error, 'delete')
  } finally {
    busyCode.value = ''
  }
}

function friendlyActionError(error, action) {
  if (error instanceof ApiError && error.status === 403) {
    return `This browser no longer owns that link, so it can't ${action} it.`
  }
  if (error instanceof ApiError && error.status === 404) {
    return 'That short link no longer exists. You can remove it from this list.'
  }
  if (error instanceof ApiError && error.status === 400) {
    return 'Enter a complete URL including http:// or https://.'
  }
  return error.message || `Could not ${action} this link.`
}

async function copyLink(link) {
  await navigator.clipboard.writeText(link.shortUrl)
  copiedCode.value = link.shortCode
  window.setTimeout(() => (copiedCode.value = ''), 1800)
}

function formatDate(value) {
  return new Intl.DateTimeFormat(undefined, {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  }).format(new Date(value))
}
</script>

<template>
  <section class="page-heading">
    <div>
      <h1>My links</h1>
    </div>
    <p>Edit or delete your saved links.</p>
  </section>

  <div class="ticks"></div>

  <section v-if="links.length" class="links-section">
    <div class="list-meta">
      <span>{{ links.length }} {{ links.length === 1 ? 'link' : 'links' }}</span>
      <RouterLink to="/">Create another <span aria-hidden="true">↗</span></RouterLink>
    </div>

    <div v-if="actionError" class="notice error list-notice" role="alert">
      <span aria-hidden="true">!</span>
      <p>{{ actionError }}</p>
    </div>

    <article v-for="link in links" :key="link.shortCode" class="link-card">
      <div class="link-card-main">
        <div class="short-link-row">
          <a class="short-link" :href="link.shortUrl" target="_blank" rel="noreferrer">
            {{ link.shortUrl }} <span aria-hidden="true">↗</span>
          </a>
          <button class="text-button" type="button" @click="copyLink(link)">
            {{ copiedCode === link.shortCode ? 'Copied!' : 'Copy' }}
          </button>
        </div>

        <form v-if="editingCode === link.shortCode" class="edit-form" @submit.prevent="saveEdit(link)">
          <label :for="`edit-${link.shortCode}`">New destination</label>
          <input
            :id="`edit-${link.shortCode}`"
            v-model="editUrl"
            type="url"
            required
            autocomplete="url"
          />
          <div class="form-actions">
            <button class="button primary compact" type="submit" :disabled="busyCode === link.shortCode">
              {{ busyCode === link.shortCode ? 'Saving…' : 'Save changes' }}
            </button>
            <button class="button ghost compact" type="button" @click="editingCode = ''">Cancel</button>
          </div>
        </form>

        <template v-else>
          <p class="destination" :title="link.longUrl">{{ link.longUrl }}</p>
          <p class="created">Created {{ formatDate(link.createdAt) }}</p>
        </template>
      </div>

      <aside class="link-stats" aria-label="Link statistics and actions">
        <div class="click-count">
          <strong v-if="counts[link.shortCode] !== undefined">{{ counts[link.shortCode] }}</strong>
          <strong v-else-if="countErrors[link.shortCode]">—</strong>
          <strong v-else class="loading-count">···</strong>
          <span>{{ countErrors[link.shortCode] || (counts[link.shortCode] === 1 ? 'click' : 'clicks') }}</span>
        </div>

        <div v-if="deletingCode === link.shortCode" class="delete-confirm">
          <p>Delete this link?</p>
          <div>
            <button
              class="text-button danger"
              type="button"
              :disabled="busyCode === link.shortCode"
              @click="confirmDelete(link)"
            >
              {{ busyCode === link.shortCode ? 'Deleting…' : 'Yes, delete' }}
            </button>
            <button class="text-button" type="button" @click="deletingCode = ''">Cancel</button>
          </div>
        </div>
        <div v-else class="card-actions">
          <button class="text-button" type="button" @click="startEditing(link)">Edit</button>
          <button class="text-button danger" type="button" @click="deletingCode = link.shortCode">
            Delete
          </button>
        </div>
      </aside>
    </article>
  </section>

  <section v-else class="empty-state">
    <span class="empty-icon" aria-hidden="true">↗</span>
    <h2>No short links yet</h2>
    <p>Your saved links will appear here.</p>
    <RouterLink class="button primary" to="/">Create a short link <span aria-hidden="true">↗</span></RouterLink>
  </section>
</template>
