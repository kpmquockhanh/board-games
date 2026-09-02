<template>
  <Toaster position="top-right" :theme="'dark'" :toast-options="{ style: { background: '#2a1f18', color: '#fdf3e4', border: '1px solid #3a2a1e' } }" />
  <AppHeader />
  <router-view />
</template>

<script setup>
import { Toaster } from 'vue-sonner'
import AppHeader from './components/AppHeader.vue'
</script>
