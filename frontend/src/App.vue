<template>
  <Toaster position="top-right" :theme="'dark'" :toast-options="{ style: { background: '#2a1f18', color: '#fdf3e4', border: '1px solid #3a2a1e' } }" />
  <AppHeader v-if="!isGameRoute" />
  <router-view />
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { Toaster } from 'vue-sonner'
import AppHeader from './components/AppHeader.vue'

const route = useRoute()
const isGameRoute = computed(() => route.path.startsWith('/exploding-kitchen'))
</script>
