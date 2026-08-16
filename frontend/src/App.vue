<script setup>
import { RouterLink, RouterView } from 'vue-router'
</script>

<template>
  <header class="site-header">
    <RouterLink class="brand" to="/" aria-label="Tiny Route home">
      <span>url shortener</span>
    </RouterLink>

    <nav aria-label="Primary navigation">
      <RouterLink to="/">Create</RouterLink>
      <RouterLink to="/links">My links</RouterLink>
    </nav>
  </header>

  <main>
    <RouterView />
  </main>

  <div class="ticks"></div>
  <footer class="site-footer">
    <p>Go · Redis · Vue</p>
    <a href="https://github.com/andrewdack/go-url-shortener" target="_blank" rel="noreferrer">
      View source <span aria-hidden="true">↗</span>
    </a>
  </footer>
</template>
