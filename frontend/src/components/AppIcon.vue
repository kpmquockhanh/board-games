<template>
  <component :is="icon" :size="size" :stroke-width="strokeWidth" />
</template>

<script setup>
defineProps({
  icon: { type: Object, required: true },
  size: { type: Number, default: 20 },
  strokeWidth: { type: Number, default: 2 },
})
</script>
