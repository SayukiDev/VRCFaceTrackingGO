<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NAlert, NButton, NButtonGroup, NCard, NCheckbox, NCollapseTransition, NEmpty, NFloatButton, NIcon, NImage, NPopconfirm, NSwitch,
  NRadioButton, NRadioGroup, NTag, NText, NSpace, NTooltip, useMessage,
} from 'naive-ui'
import {
  Add24Regular, ArrowUp24Regular, Checkmark20Regular, ArrowDown24Regular, ArrowSync24Regular, ChevronDown20Regular, Delete24Regular,
  Info16Regular, Info24Regular, Open16Regular, Person16Regular,
} from '@vicons/fluent'
import {
  GetDriverCatalog, GetModuleCatalog, GetModuleConfigDirty, InstallModule, OpenURL, ReloadModules, SetModuleEnabled, SetModuleOrder,
  SetModuleUse, UninstallModule,
} from '@go/app/app'
import type * as router from '@go/internal/router'
import type * as service from '@go/internal/service'
import { Events, useEvent } from '../composables/useBackend'
import logo from '../assets/icon.png'
import InfoCard from '../components/InfoCard.vue'

const { t } = useI18n()
const message = useMessage()
const modules = ref<router.Registered[]>([])
// Drivers are listed on their own: they are never loaded, their modules are.
const drivers = ref<service.Driver[]>([])
const tab = ref<'modules' | 'drivers'>('modules')

// Cards without icons show the app logo so their text lines up with the rest.
function iconsOf(icons: string[]) {
  return icons.length > 0 ? icons : [logo]
}
// dirty: the saved list differs from what is running. reloading: a reload
// is in flight (cleared when the router reports the new module list).
const dirty = ref(false)
const reloading = ref(false)
// IDs of the cards whose details are open.
const expanded = ref(new Set<string>())

function toggle(id: string) {
  if (!expanded.value.delete(id)) expanded.value.add(id)
}

async function refresh() {
  ;[modules.value, drivers.value, dirty.value] = await Promise.all([GetModuleCatalog(), GetDriverCatalog(), GetModuleConfigDirty()])
}

function driverName(id: string) {
  return drivers.value.find((d) => d.id === id)?.name ?? id
}
refresh()
useEvent(Events.modules, () => {
  reloading.value = false
  refresh()
})

async function setEnabled(m: router.Registered, enabled: boolean) {
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
async function setUse(m: router.Registered, eye: boolean, expression: boolean) {
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

async function install() {
  try {
    const name = await InstallModule()
    if (!name) return
    message.success(t('modules.installed', { name }))
    await refresh()
    // Show what was installed: a driver lives on its own tab.
    tab.value = drivers.value.some((d) => d.name === name) ? 'drivers' : 'modules'
  } catch (e) {
    message.error(String(e))
  }
}

// Modules inside a driver cannot be uninstalled on their own; only the
// driver (m is then a driver) can.
async function uninstall(m: { id: string; name: string }) {
  try {
    await UninstallModule(m.id)
    message.success(t('modules.uninstalled', { name: m.name }))
    await refresh()
  } catch (e) {
    message.error(String(e))
  }
}

async function reload() {
  reloading.value = true
  dirty.value = false
  await ReloadModules()
}

function status(m: router.Registered): { text: string; type: 'success' | 'warning' | 'default' } {
  if (!m.enabled) return { text: t('modules.disabled'), type: 'default' }
  if (!m.loaded) return { text: t('modules.notLoaded'), type: 'warning' }
  return { text: m.active ? t('modules.active') : t('modules.idle'), type: 'success' }
}
</script>

<template>
  <div class="stack with-fab">
    <n-radio-group v-model:value="tab" class="segmented">
      <n-radio-button v-for="v in (['modules', 'drivers'] as const)" :key="v" :value="v">
        <span class="segment">
          <n-icon v-if="tab === v" :size="18"><checkmark20-regular /></n-icon>
          {{ v === 'modules' ? t('modules.tabModules') : t('modules.tabDrivers') }}
        </span>
      </n-radio-button>
    </n-radio-group>

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

    <template v-if="tab === 'modules'">
      <info-card :icon="Info24Regular"><n-text depth="3">{{ t('modules.intro') }}</n-text></info-card>
      <n-empty v-if="modules.length === 0" :description="t('modules.none')" />

      <n-card v-for="(m, i) in modules" :key="m.id" size="small">
        <div class="row">
          <div class="summary" @click="toggle(m.id)">
            <n-icon class="chevron" :class="{ open: expanded.has(m.id) }" size="18"><chevron-down20-regular /></n-icon>
            <div class="icons">
              <n-image v-for="(src, k) in iconsOf(m.icons)" :key="k" :src="src" height="40" preview-disabled object-fit="cover" />
            </div>
            <div class="info">
              <div class="title">
                <span class="name">{{ m.name }}</span>
                <n-text v-if="m.version" depth="3" class="small version">v{{ m.version }}</n-text>
              </div>
              <n-text v-if="m.description" depth="3" class="desc" :class="{ clamp: !expanded.has(m.id) }">{{ m.description }}</n-text>
            </div>
          </div>
          <n-button-group class="order">
            <n-button quaternary :disabled="i === 0 || reloading" @click="move(i, -1)">
              <template #icon><n-icon><arrow-up24-regular /></n-icon></template>
            </n-button>
            <n-button quaternary :disabled="i === modules.length - 1 || reloading" @click="move(i, 1)">
              <template #icon><n-icon><arrow-down24-regular /></n-icon></template>
            </n-button>
          </n-button-group>
          <n-switch class="enabled" :value="m.enabled" :disabled="reloading" @update:value="(v: boolean) => setEnabled(m, v)">
            <template #checked>{{ t('modules.enabled') }}</template>
            <template #unchecked>{{ t('modules.disabled') }}</template>
          </n-switch>
          <!-- A plain span is the tooltip trigger: Naive popovers cannot use
               another popover component (the popconfirm) as their trigger. -->
          <n-tooltip v-if="!m.driver">
            <template #trigger>
              <span class="uninstall">
                <n-popconfirm @positive-click="uninstall(m)">
                  <template #trigger>
                    <n-button quaternary type="error" :aria-label="t('modules.uninstall')" :disabled="reloading">
                      <template #icon><n-icon><delete24-regular /></n-icon></template>
                    </n-button>
                    </template>
                  {{ t('modules.uninstallConfirm', { name: m.name }) }}
                </n-popconfirm>
              </span>
              </template>
            {{ t('modules.uninstall') }}
          </n-tooltip>
        </div>

        <n-collapse-transition :show="expanded.has(m.id)">
          <div class="details">
            <div v-if="m.author || m.pageUrl" class="meta">
              <span v-if="m.author" class="author">
                <n-icon :size="16"><person16-regular /></n-icon>
                <n-text depth="3">{{ m.author }}</n-text>
              </span>
              <n-button v-if="m.pageUrl" text type="primary" size="small" @click="OpenURL(m.pageUrl)">
                <template #icon><n-icon><open16-regular /></n-icon></template>
                {{ t('modules.website') }}
              </n-button>
            </div>
            <div v-if="m.usageInstructions" class="usage">
              <n-icon class="usage-icon"><info16-regular /></n-icon>
              <n-text depth="3">{{ m.usageInstructions }}</n-text>
            </div>
          </div>
        </n-collapse-transition>

        <div class="bar">
          <n-space size="small" align="center">
            <n-tag size="small" :bordered="false" :type="status(m).type">{{ status(m).text }}</n-tag>
            <n-tag v-if="m.driver" size="small" :bordered="false">{{ t('modules.viaDriver', { name: driverName(m.driver) }) }}</n-tag>
            <n-tag v-if="m.supportsEye" size="small" :bordered="false" :type="m.usingEye ? 'info' : 'default'">{{ t('modules.eye') }}</n-tag>
            <n-tag v-if="m.supportsExpression" size="small" :bordered="false" :type="m.usingExpression ? 'info' : 'default'">{{ t('modules.face') }}</n-tag>
          </n-space>
          <n-space size="medium" align="center" class="use">
            <n-text depth="3" class="small">{{ t('modules.useFor') }}</n-text>
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
      </n-card>
    </template>

    <template v-else>
      <info-card :icon="Info24Regular"><n-text depth="3">{{ t('modules.driversIntro') }}</n-text></info-card>
      <n-empty v-if="drivers.length === 0" :description="t('modules.noDrivers')" />
      <n-card v-for="d in drivers" :key="d.id" size="small">
        <div class="row">
          <div class="summary" @click="toggle(d.id)">
            <n-icon class="chevron" :class="{ open: expanded.has(d.id) }" size="18"><chevron-down20-regular /></n-icon>
            <div class="icons">
              <n-image v-for="(src, k) in iconsOf(d.icons)" :key="k" :src="src" height="40" preview-disabled object-fit="cover" />
            </div>
            <div class="info">
              <div class="title">
                <span class="name">{{ d.name }}</span>
                <n-text v-if="d.version" depth="3" class="small version">v{{ d.version }}</n-text>
              </div>
              <n-text v-if="d.description" depth="3" class="desc" :class="{ clamp: !expanded.has(d.id) }">{{ d.description }}</n-text>
            </div>
          </div>
          <n-tooltip>
            <template #trigger>
              <span class="uninstall">
                <n-popconfirm @positive-click="uninstall(d)">
                  <template #trigger>
                    <n-button quaternary type="error" :aria-label="t('modules.uninstall')" :disabled="reloading">
                      <template #icon><n-icon><delete24-regular /></n-icon></template>
                    </n-button>
                    </template>
                  {{ t('modules.uninstallDriverConfirm', { name: d.name }) }}
                </n-popconfirm>
              </span>
              </template>
            {{ t('modules.uninstall') }}
          </n-tooltip>
        </div>

        <n-collapse-transition :show="expanded.has(d.id)">
          <div class="details">
            <div v-if="d.author || d.pageUrl" class="meta">
              <span v-if="d.author" class="author">
                <n-icon :size="16"><person16-regular /></n-icon>
                <n-text depth="3">{{ d.author }}</n-text>
              </span>
              <n-button v-if="d.pageUrl" text type="primary" size="small" @click="OpenURL(d.pageUrl)">
                <template #icon><n-icon><open16-regular /></n-icon></template>
                {{ t('modules.website') }}
              </n-button>
            </div>
            <div v-if="d.usageInstructions" class="usage">
              <n-icon class="usage-icon"><info16-regular /></n-icon>
              <n-text depth="3">{{ d.usageInstructions }}</n-text>
            </div>
          </div>
        </n-collapse-transition>
      </n-card>
    </template>

    <Teleport defer to="#page-actions">
      <n-tooltip>
        <template #trigger>
          <n-button quaternary circle size="large" class="reload" :loading="reloading" :aria-label="t('modules.reload')" @click="reload">
            <template #icon><n-icon :size="24"><arrow-sync24-regular /></n-icon></template>
          </n-button>
        </template>
        {{ t('modules.reload') }}
      </n-tooltip>
    </Teleport>
    <n-tooltip placement="left">
      <template #trigger>
        <n-float-button
          type="primary"
          shape="square"
          :width="56"
          :height="56"
          :right="24"
          :bottom="24"
          :aria-label="t('modules.install')"
          @click="!reloading && install()"
        >
          <n-icon :size="24"><add24-regular /></n-icon>
        </n-float-button>
      </template>
      {{ t('modules.install') }}
    </n-tooltip>
  </div>
</template>

<style scoped>
/* 48px icon button with a 24px icon, matching the M3 minimum touch target. */
.reload {
  width: 48px;
  height: 48px;
}
/* Keep the last card clear of the FAB (56px + 24px margin). */
.with-fab {
  padding-bottom: 80px;
}
.tip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
/* Everything lines up with the first line, so a wrapped description of an
   open card grows downwards without moving the controls. */
.row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 4px 8px;
}
.summary {
  flex: 1;
  /* Let the description shrink (ellipsis) instead of the controls. */
  min-width: 0;
  display: flex;
  align-items: flex-start;
  gap: 16px;
  margin-right: 8px;
  cursor: pointer;
  user-select: none;
}
.chevron {
  margin-top: 11px;
  transition: transform 0.2s;
  transform: rotate(-90deg);
}
.chevron.open {
  transform: rotate(0deg);
}
.icons {
  display: flex;
  gap: 8px;
}
.chevron,
.icons,
.order,
.enabled {
  flex-shrink: 0;
}
.order {
  margin-top: 4px;
}
.enabled {
  margin-top: 9px;
}
.info {
  flex: 1;
  min-width: 0;
}
.title {
  display: flex;
  align-items: baseline;
  gap: 8px;
  min-width: 0;
}
.name {
  font-size: var(--md-sys-typescale-title-medium-size);
  line-height: var(--md-sys-typescale-title-medium-line-height);
  font-weight: var(--md-sys-typescale-title-medium-weight);
  letter-spacing: var(--md-sys-typescale-title-medium-tracking);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.version {
  white-space: nowrap;
  flex-shrink: 0;
}
.desc {
  display: block;
  font-size: 13px;
}
/* One line while collapsed; the full text shows when the card is open. */
.desc.clamp {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
/* Details and bar line up with the module name: chevron, icon and their
   gaps. */
.details {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px 8px 0 98px;
}
.bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px 24px;
  padding: 10px 8px 4px 98px;
}
/* M3 segmented button: equal segments, a check mark on the selected one. */
.segmented {
  display: flex;
  width: fit-content;
}
.segmented :deep(.n-radio-button) {
  min-width: 140px;
  text-align: center;
}
.segmented :deep(.n-radio-button:not(.n-radio-button--checked):hover) {
  background: var(--md-sys-color-on-surface-hover);
}
.segment {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-weight: 500;
}
.uninstall {
  display: inline-flex;
  flex-shrink: 0;
  margin-top: 4px;
}
.meta {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 12px;
}
.author {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--md-sys-color-on-surface-variant);
}
.usage {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  font-size: 12px;
}
.usage-icon {
  margin-top: 2px;
  flex-shrink: 0;
}
.small {
  font-size: 12px;
}
</style>
