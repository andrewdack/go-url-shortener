<script setup>
import { ref } from 'vue'
import { ApiError, createShortUrl } from '../services/api'
import { getUserId, rememberLink } from '../services/storage'

const longUrl = ref('')
const result = ref(null)
const errorMessage = ref('')
const submitting = ref(false)
const copied = ref(false)

function friendlyError(error) {
  if (error instanceof ApiError && error.status === 429) {
    return "You're creating links too quickly. Give it a minute, then try again."
  }
  if (error instanceof ApiError && error.status === 400) {
    return 'Enter a complete URL including http:// or https://.'
  }
  return error.message || 'Something went wrong. Please try again.'
}

async function submit() {
  errorMessage.value = ''
  result.value = null
  copied.value = false
  submitting.value = true

  try {
    const payload = await createShortUrl(longUrl.value.trim(), getUserId())
    result.value = payload
    rememberLink({ shortUrl: payload.shortUrl, longUrl: longUrl.value.trim() })
  } catch (error) {
    errorMessage.value = friendlyError(error)
  } finally {
    submitting.value = false
  }
}

async function copyResult() {
  await navigator.clipboard.writeText(result.value.shortUrl)
  copied.value = true
  window.setTimeout(() => (copied.value = false), 1800)
}
</script>

<template>
  <section class="hero-section">
    <p class="hero-copy">Paste a URL and get a short link.</p>

    <form class="create-form" @submit.prevent="submit">
      <div class="input-row">
        <input
          id="long-url"
          v-model="longUrl"
          type="url"
          required
          inputmode="url"
          autocomplete="url"
          placeholder="https://example.com/long-path"
        />
        <button class="button primary" type="submit" :disabled="submitting">
          {{ submitting ? 'Shortening…' : 'Shorten link' }}
          <span aria-hidden="true">↗</span>
        </button>
      </div>
      <p class="field-hint">Use the full URL, including <code>https://</code></p>
    </form>

    <div v-if="errorMessage" class="notice error" role="alert">
      <span aria-hidden="true">!</span>
      <p>{{ errorMessage }}</p>
    </div>

    <div v-if="result" class="result-card" aria-live="polite">
      <div>
        <span class="result-label">Short link</span>
        <a :href="result.shortUrl" target="_blank" rel="noreferrer">{{ result.shortUrl }}</a>
      </div>
      <button class="button secondary" type="button" @click="copyResult">
        {{ copied ? 'Copied!' : 'Copy link' }}
      </button>
    </div>
  </section>

</template>
