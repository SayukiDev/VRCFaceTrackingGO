<script setup lang="ts">
import { NCard, NIcon } from 'naive-ui'

defineProps<{ icon?: any }>()
</script>

<template>
  <n-card size="small" :bordered="true">
    <div class="row">
      <n-icon v-if="icon" size="22" class="icon"><component :is="icon" /></n-icon>
      <div class="body"><slot /></div>
      <div class="aside"><slot name="aside" /></div>
    </div>
  </n-card>
</template>

<style scoped>
.row {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 4px 6px;
}
.icon {
  flex: none;
  margin: 0 10px;
}
.body {
  flex: 1;
  min-width: 0;
}
.aside {
  flex: none;
  text-align: right;
  margin-right: 16px;
}
</style>
