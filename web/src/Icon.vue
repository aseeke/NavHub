<script setup>
defineProps({ name: String })
const paths = {
  menu: ['M4 4h6v6H4zM14 4h6v6h-6zM4 14h6v6H4zM14 14h6v6h-6z'],
  search: ['M10.5 17a6.5 6.5 0 1 0 0-13 6.5 6.5 0 0 0 0 13', 'm16 16 4 4'],
  plus: ['M12 5v14M5 12h14'],
  close: ['m6 6 12 12M18 6 6 18'],
  edit: ['m14 5 5 5M4 20l4-1 12-12a2.1 2.1 0 0 0-3-3L5 16l-1 4Z'],
  settings: ['M9 3h6l1 3 3 1 2 5-2 5-3 1-1 3H9l-1-3-3-1-2-5 2-5 3-1 1-3Z', 'M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6'],
  grip: ['M9 5h.01M15 5h.01M9 12h.01M15 12h.01M9 19h.01M15 19h.01'],
  folder: ['M3 7V5a1 1 0 0 1 1-1h5l2 3h9a1 1 0 0 1 1 1v11a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V7Z'],
  check: ['m5 12 4 4L19 6'],
  reset: ['M4 10a8 8 0 1 1 1 7', 'M4 4v6h6'],
  chevron: ['m8 10 4 4 4-4'],
}
</script>
<template>
  <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path v-for="(d, index) in paths[name] || paths.menu" :key="index" :d="d"/></svg>
</template>
