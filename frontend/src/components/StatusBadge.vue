<script setup>
import { computed } from 'vue'
const props = defineProps({ value: { type: String, required: true }, kind: { type: String, default: 'status' } })
const label = computed(() => (props.value || '').replace('_', ' '))
const tone = computed(() => `${props.kind}-${props.value}`)
</script>

<template><span class="badge" :class="tone"><span class="badge-dot"></span>{{ label }}</span></template>
