<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NAlert, NButton, NButtonGroup, NCard, NCheckbox, NIcon, NImage, NSwitch, NTag, NText, NSpace, NTooltip, useMessage,
} from 'naive-ui'
import { ArrowUp24Regular, ArrowDown24Regular, ArrowSync24Regular } from '@vicons/fluent'
import {
  GetModuleCatalog, GetModuleConfigDirty, ReloadModules, SetModuleEnabled, SetModuleOrder, SetModuleUse,
} from '../../wailsjs/go/app/App'
import type { manager } from '../../wailsjs/go/models'
import { Events, useEvent } from '../composables/useBackend'

const { t } = useI18n()
const message = useMessage()
const modules = ref<manager.Registered[]>([])
// dirty: the saved list differs from what is running. reloading: a reload
// is in flight (cleared when the manager reports the new module list).
const dirty = ref(false)
const reloading = ref(false)

async function refresh() {
  ;[modules.value, dirty.value] = await Promise.all([GetModuleCatalog(), GetModuleConfigDirty()])
}
refresh()
useEvent(Events.modules, () => {
  reloading.value = false
  refresh()
})

async function setEnabled(m: manager.Registered, enabled: boolean) {
  m.enabled = enabled
  try {
    await SetModuleEnabled(m.id, enabled)
    dirty.value = true
  } catch (e) {
    message.error(String(e))
    await refresh()
  }
}

async function move(index: number, delta: number) {
  const target = index + delta
  if (target < 0 || target >= modules.value.length) return
  const order = modules.value.map((m) => m.id)
  ;[order[index], order[target]] = [order[target], order[index]]
  try {
    await SetModuleOrder(order)
    await refresh()
  } catch (e) {
    message.error(String(e))
  }
}

// "Use for" checkboxes decide what the module may claim at initialization;
// like ordering they need a reload.
async function setUse(m: manager.Registered, eye: boolean, expression: boolean) {
  m.useEye = eye
  m.useExpression = expression
  try {
    await SetModuleUse(m.id, eye, expression)
    dirty.value = true
  } catch (e) {
    message.error(String(e))
    await refresh()
  }
}

async function reload() {
  reloading.value = true
  dirty.value = false
  await ReloadModules()
}

function status(m: manager.Registered): { text: string; type: 'success' | 'warning' | 'default' } {
  if (!m.enabled) return { text: t('modules.disabled'), type: 'default' }
  if (!m.loaded) return { text: t('modules.notLoaded'), type: 'warning' }
  return { text: m.active ? t('modules.active') : t('modules.idle'), type: 'success' }
}
</script>

<template>
  <div class="stack">
    <n-text depth="3">{{ t('modules.intro') }}</n-text>

    <n-alert v-if="reloading" type="info" :title="t('modules.reloadingTitle')">
      {{ t('modules.reloadingBody') }}
    </n-alert>
    <n-alert v-else-if="dirty" type="warning" :title="t('modules.dirtyTitle')">
      <div class="tip">
        <span>{{ t('modules.dirtyBody') }}</span>
        <n-button type="primary" size="small" @click="reload">
          <template #icon><n-icon><arrow-sync24-regular /></n-icon></template>
          {{ t('modules.reload') }}
        </n-button>
      </div>
    </n-alert>

    <n-card v-for="(m, i) in modules" :key="m.id" size="small">
      <div class="row">
        <div class="rank"><n-text depth="3">{{ i + 1 }}</n-text></div>
        <div class="icons">
          <n-image v-for="(src, k) in m.icons" :key="k" :src="src" height="64" preview-disabled object-fit="cover" />
        </div>
        <div class="info">
          <div class="name">{{ m.name }}</div>
          <n-space size="small" align="center">
            <n-tag size="small" :bordered="false" :type="status(m).type">{{ status(m).text }}</n-tag>
            <n-tag v-if="m.supportsEye" size="small" :bordered="false" :type="m.usingEye ? 'info' : 'default'">{{ t('modules.eye') }}</n-tag>
            <n-tag v-if="m.supportsExpression" size="small" :bordered="false" :type="m.usingExpression ? 'info' : 'default'">{{ t('modules.face') }}</n-tag>
          </n-space>
          <n-space size="medium" align="center" class="use">
            <n-text depth="3" class="use-label">{{ t('modules.useFor') }}</n-text>
            <n-tooltip v-if="m.supportsEye">
              <template #trigger>
                <n-checkbox :checked="m.useEye" :disabled="!m.enabled || reloading" @update:checked="(v: boolean) => setUse(m, v, m.useExpression)">
                  {{ t('modules.eye') }}
                </n-checkbox>
              </template>
              {{ t('modules.useForHint') }}
            </n-tooltip>
            <n-tooltip v-if="m.supportsExpression">
              <template #trigger>
                <n-checkbox :checked="m.useExpression" :disabled="!m.enabled || reloading" @update:checked="(v: boolean) => setUse(m, m.useEye, v)">
                  {{ t('modules.face') }}
                </n-checkbox>
              </template>
              {{ t('modules.useForHint') }}
            </n-tooltip>
          </n-space>
        </div>
        <n-button-group class="order">
          <n-button quaternary :disabled="i === 0 || reloading" @click="move(i, -1)">
            <template #icon><n-icon><arrow-up24-regular /></n-icon></template>
          </n-button>
          <n-button quaternary :disabled="i === modules.length - 1 || reloading" @click="move(i, 1)">
            <template #icon><n-icon><arrow-down24-regular /></n-icon></template>
          </n-button>
        </n-button-group>
        <n-switch :value="m.enabled" :disabled="reloading" @update:value="(v: boolean) => setEnabled(m, v)">
          <template #checked>{{ t('modules.enabled') }}</template>
          <template #unchecked>{{ t('modules.disabled') }}</template>
        </n-switch>
      </div>
    </n-card>
  </div>
</template>

<style scoped>
.tip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.row {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 6px 8px;
}
.rank {
  width: 20px;
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.icons {
  display: flex;
  gap: 8px;
}
.info {
  flex: 1;
}
.name {
  font-size: 20px;
  font-weight: 600;
  margin-bottom: 6px;
}
.use {
  margin-top: 8px;
}
.use-label {
  font-size: 12px;
}
.order {
  margin-right: 8px;
}
</style>
