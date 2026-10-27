<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { NCard, NImage, NSwitch, NTag, NText } from 'naive-ui'
import type * as router from '@go/internal/router'
import { SetModuleActive } from '@go/app/app'

defineProps<{ module: router.ModuleInfo }>()
const { t } = useI18n()

function toggle(id: string, value: boolean) {
  void SetModuleActive(id, value)
}
</script>

<template>
  <n-card size="small">
    <div class="row">
      <div class="icons">
        <n-image v-for="(src, i) in module.icons" :key="i" :src="src" height="150" preview-disabled object-fit="cover" />
      </div>
      <div class="info">
        <div class="name">{{ module.name }}</div>
        <template v-if="module.usingEye">
          <n-text depth="3" :class="{ inactive: !module.eyeActive }">
            {{ module.eyeActive ? t('home.eyeActive') : t('home.eyeInactive') }}
          </n-text>
          <br />
        </template>
        <n-text v-if="module.usingExpression" depth="3" :class="{ inactive: !module.expressionActive }">
          {{ module.expressionActive ? t('home.faceActive') : t('home.faceInactive') }}
        </n-text>
        <div v-if="module.stale">
          <n-tag size="small" type="warning" :bordered="false">{{ t('home.noData') }}</n-tag>
        </div>
      </div>
      <div class="toggle">
        <n-switch :value="module.active" size="large" @update:value="(v: boolean) => toggle(module.id, v)">
          <template #checked>{{ t('home.toggleTracking') }}</template>
          <template #unchecked>{{ t('home.toggleTracking') }}</template>
        </n-switch>
      </div>
    </div>
  </n-card>
</template>

<style scoped>
.row {
  display: flex;
  align-items: center;
  gap: 24px;
  padding: 8px;
}
.icons {
  display: flex;
  gap: 8px;
}
.info {
  flex: 1;
}
.name {
  font-size: var(--md-sys-typescale-headline-small-size);
  line-height: var(--md-sys-typescale-headline-small-line-height);
  font-weight: var(--md-sys-typescale-headline-small-weight);
  letter-spacing: var(--md-sys-typescale-headline-small-tracking);
  margin-bottom: 6px;
}
.inactive {
  opacity: 0.5;
}
.toggle {
  margin-right: 16px;
}
</style>
