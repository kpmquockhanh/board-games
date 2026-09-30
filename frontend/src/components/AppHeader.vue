<template>
  <nav class="app-header">
    <router-link to="/" class="header-brand">
      <BrandMark />
      <span class="brand-name">Ping</span>
    </router-link>

    <div class="breadcrumb" v-if="currentGame">
      <router-link to="/" class="crumb">Home</router-link>
      <span class="separator">›</span>
      <span class="crumb current">{{ currentGame.name }}</span>
    </div>
    <div class="breadcrumb" v-else>
      <span class="crumb current">Home</span>
    </div>

    <span class="account-chip" v-if="signedIn">{{ account.user.display_name || 'Signed in' }}</span>

    <button class="menu-toggle" @click="menuOpen = !menuOpen" :class="{ open: menuOpen }">
      <Menu :size="20" />
    </button>

    <div class="menu-dropdown" v-if="menuOpen" @click="menuOpen = false">
      <router-link to="/" class="menu-item" :class="{ active: !currentGame }">Home</router-link>
      <router-link
        v-for="game in gameList"
        :key="game.route"
        :to="game.route"
        class="menu-item"
        :class="{ active: currentGame?.code === game.code }"
      >{{ game.name }}</router-link>

      <template v-if="signedIn">
        <div class="menu-divider"></div>
        <div class="menu-note">Signed in with {{ account.user.providers.map(label).join(', ') }}</div>
        <button class="menu-item" @click="signOut">Sign out</button>
      </template>
      <template v-else-if="account?.providers?.length">
        <div class="menu-divider"></div>
        <a
          v-for="p in account.providers"
          :key="p"
          :href="loginUrl(p, route.fullPath)"
          class="menu-item"
        >Sign in with {{ label(p) }}</a>
      </template>
    </div>
  </nav>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { Menu } from '@lucide/vue'
import BrandMark from './BrandMark.vue'
import { GAMES } from '../config/games.js'
import { getAuthSession, loginUrl, logout } from '../api.js'

const route = useRoute()
const menuOpen = ref(false)

const gameList = Object.values(GAMES)

const currentGame = computed(() => {
  return gameList.find(g => g.route === route.path) || null
})

watch(() => route.path, () => {
  menuOpen.value = false
})

// Signing in is optional: a guest plays just the same. It only keeps who you
// are once this browser forgets, and across devices.
const account = ref(null)
const signedIn = computed(() => account.value?.user && !account.value.user.guest)
const LABELS = { google: 'Google', discord: 'Discord' }
const label = (p) => LABELS[p] || p

async function refreshAccount() {
  account.value = await getAuthSession()
}

async function signOut() {
  await logout()
  await refreshAccount()
}

onMounted(refreshAccount)
</script>

<style scoped>
.app-header {
  position: sticky;
  top: 0;
  z-index: 100;
  display: flex;
  align-items: center;
  height: 52px;
  padding: 0 20px;
  background: var(--panel);
  border-bottom: 1px solid var(--line);
}

.header-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  text-decoration: none;
  color: inherit;
  flex-shrink: 0;
}

.brand-name {
  font-family: 'Baloo 2', sans-serif;
  font-weight: 800;
  font-size: 1.15rem;
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: 24px;
  font-size: 0.85rem;
  color: var(--mild-cream);
  opacity: 0.6;
}

.crumb {
  text-decoration: none;
  color: inherit;
  transition: opacity 0.15s ease;
}

a.crumb:hover {
  opacity: 1;
}

.crumb.current {
  opacity: 1;
  color: var(--steam-cream);
  font-weight: 600;
}

.separator {
  opacity: 0.4;
}

.menu-toggle {
  margin-left: auto;
  background: none;
  border: 1px solid transparent;
  color: var(--steam-cream);
  font-size: 1.2rem;
  padding: 6px 10px;
  border-radius: 8px;
  cursor: pointer;
  transition: border-color 0.15s ease, background 0.15s ease;
}

.menu-toggle:hover {
  border-color: var(--line);
  background: rgba(253, 243, 228, 0.05);
}

.menu-dropdown {
  position: absolute;
  top: 52px;
  right: 16px;
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 12px;
  padding: 6px;
  min-width: 180px;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.4);
}

.menu-item {
  display: block;
  padding: 10px 14px;
  border-radius: 8px;
  font-size: 0.88rem;
  color: var(--mild-cream);
  text-decoration: none;
  transition: background 0.12s ease;
}

.menu-item:hover {
  background: rgba(253, 243, 228, 0.08);
  color: var(--steam-cream);
}

button.menu-item {
  width: 100%;
  text-align: left;
  background: none;
  border: none;
  font: inherit;
  cursor: pointer;
}

.menu-divider {
  height: 1px;
  margin: 6px 4px;
  background: var(--line);
}

.menu-note {
  padding: 6px 14px 2px;
  font-size: 0.75rem;
  color: var(--mild-cream);
  opacity: 0.6;
}

.account-chip {
  margin-left: auto;
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--steam-cream);
}

.account-chip + .menu-toggle {
  margin-left: 8px;
}

.menu-item.active {
  color: var(--chili-orange);
  font-weight: 600;
}

@media (max-width: 600px) {
  .breadcrumb {
    display: none;
  }
}
</style>
