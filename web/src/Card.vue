<script setup>
import { initials, ink } from './utils'
import { ref, watch } from 'vue'
const props = defineProps({ item: Object, settings: Object, preview: Boolean, managing: Boolean })
const failed = ref(false)
watch(() => props.item.icon, () => failed.value = false)
defineEmits(['edit'])
</script>
<template>
  <component :is="preview || managing ? 'button' : 'a'" class="nav-card" :class="{ editable: managing }" :type="preview || managing ? 'button' : undefined" :href="!preview && !managing ? item.url : undefined" :target="item.target || settings.target" rel="noopener noreferrer" :style="{ background: item.color || settings.cardColor, color: ink(item.color || settings.cardColor) }" @click="managing && $emit('edit', item)">
    <span class="card-icon" :style="{ background: item.iconColor || settings.iconColor, color: ink(item.iconColor || settings.iconColor) }">
      <img v-if="item.icon && !failed" :key="item.icon" :src="item.icon" alt="" loading="lazy" referrerpolicy="no-referrer" @error="failed = true">
      <span v-else>{{ initials(item.name) }}</span>
    </span>
    <span class="card-name">{{ item.name || '项目名称' }}</span><span v-if="managing" class="edit-mark" aria-hidden="true">✎</span>
  </component>
</template>
