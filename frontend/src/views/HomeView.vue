<script setup>
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
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
    <div class="eyebrow"><span class="status-dot"></span> URL shortener</div>
    <h1>Long story,<br /><span>short link.</span></h1>
    <p class="hero-copy">
      Turn unwieldy URLs into tidy, shareable links. No account needed—your browser remembers
      what you make.
    </p>

    <form class="create-form" @submit.prevent="submit">
      <label for="long-url">Paste a long URL</label>
      <div class="input-row">
        <input
          id="long-url"
          v-model="longUrl"
          type="url"
          required
          inputmode="url"
          autocomplete="url"
          placeholder="https://example.com/a/very/long/path"
        />
        <button class="button primary" type="submit" :disabled="submitting">
          {{ submitting ? 'Shortening…' : 'Shorten link' }}
          <span aria-hidden="true">↗</span>
        </button>
      </div>
      <p class="field-hint">Include the full address, starting with <code>https://</code></p>
    </form>

    <div v-if="errorMessage" class="notice error" role="alert">
      <span aria-hidden="true">!</span>
      <p>{{ errorMessage }}</p>
    </div>

    <div v-if="result" class="result-card" aria-live="polite">
      <div>
        <span class="result-label">Your short link is ready</span>
        <a :href="result.shortUrl" target="_blank" rel="noreferrer">{{ result.shortUrl }}</a>
      </div>
      <button class="button secondary" type="button" @click="copyResult">
        {{ copied ? 'Copied!' : 'Copy link' }}
      </button>
    </div>
  </section>

  <div class="ticks"></div>

  <section class="feature-grid">
    <article>
      <span class="step">01</span>
      <h2>Paste it</h2>
      <p>Drop in any valid web address. We’ll handle the long part.</p>
    </article>
    <article>
      <span class="step">02</span>
      <h2>Share it</h2>
      <p>Copy your eight-character short link and send it anywhere.</p>
    </article>
    <article>
      <span class="step">03</span>
      <h2>Manage it</h2>
      <p>
        Edit destinations, view clicks, or remove links from
        <RouterLink to="/links">My links</RouterLink>.
      </p>
    </article>
  </section>
</template>
