<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NText, NAlert, NButton, NIcon } from 'naive-ui'
import { PlugConnected24Regular, Person24Regular, ArrowSync24Regular } from '@vicons/fluent'
import { GetModules, GetOscStatus, GetAvatar, GetParameterStats, ReloadModules } from '../../wailsjs/go/app/App'
import type { router, service, oscquery, sender } from '../../wailsjs/go/models'
import { Events, useBackendState, useEvent } from '../composables/useBackend'
import InfoCard from '../components/InfoCard.vue'
import ModuleCard from '../components/ModuleCard.vue'
import TrackingAssignment from '../components/TrackingAssignment.vue'

const { t } = useI18n()
const modules = useBackendState<router.ModuleInfo[]>(GetModules, Events.modules, [])
const osc = useBackendState<service.OscStatus | null>(GetOscStatus, Events.oscStats, null)
const avatar = useBackendState<oscquery.AvatarInfo | null>(GetAvatar, Events.avatar, null)
const stats = useBackendState<sender.ParameterStats | null>(GetParameterStats, Events.parameters, null)

// Modules that are running but have not delivered new data for a while.
const stale = computed(() => modules.value.filter((m) => m.stale))
// reloading: a reload is in flight (cleared by the next module list update).
const reloading = ref(false)
useEvent(Events.modules, () => (reloading.value = false))

async function reload() {
  reloading.value = true
  await ReloadModules()
}
</script>

<template>
  <div class="stack">
    <n-alert v-if="stale.length > 0" :title="t('home.staleTitle')" type="warning">
      <div class="tip">
        <span>{{ t('home.staleBody', { names: stale.map((m) => m.name).join(', ') }) }}</span>
        <n-button type="primary" size="small" :loading="reloading" @click="reload">
          <template #icon><n-icon><arrow-sync24-regular /></n-icon></template>
          {{ t('modules.reload') }}
        </n-button>
      </div>
    </n-alert>

    <module-card v-for="m in modules" :key="m.id" :module="m" />

    <tracking-assignment />

    <n-alert v-if="modules.length === 0" :title="t('home.noModulesTitle')" type="warning">
      <div class="tip">
        <span>{{ t('home.noModulesBody') }}</span>
        <n-button type="primary" size="small" :loading="reloading" @click="reload">
          <template #icon><n-icon><arrow-sync24-regular /></n-icon></template>
          {{ t('modules.reload') }}
        </n-button>
      </div>
    </n-alert>

    <info-card :icon="PlugConnected24Regular">
      <div class="subtitle">{{ t('home.oscStatus') }}</div>
      <div v-if="osc?.connected"><n-text depth="3">{{ t('home.listeningOn', { port: osc?.listenPort }) }}</n-text></div>
      <div><n-text depth="3">{{ t('home.sendingOn', { port: osc?.sendPort }) }}</n-text></div>
      <template #aside>
        <div><n-text depth="3">{{ t('home.msgsIn', { n: osc?.messagesIn ?? 0 }) }}</n-text></div>
        <div><n-text depth="3">{{ t('home.msgsOut', { n: osc?.messagesOut ?? 0 }) }}</n-text></div>
      </template>
    </info-card>

    <info-card :icon="Person24Regular">
      <div class="subtitle">{{ t('home.avatarStatus') }}</div>
      <div><n-text depth="3">{{ avatar?.name ?? t('home.loading') }}</n-text></div>
      <div><n-text depth="3">{{ avatar?.id ?? t('home.loading') }}</n-text></div>
      <template #aside>
        <div><n-text depth="3">{{ t('home.parameters', { n: stats?.count ?? 0 }) }}</n-text></div>
        <div v-if="(stats?.deprecated ?? 0) > 0">
          <span class="dot" /><n-text depth="3">{{ t('home.legacyParameters', { n: stats?.deprecated }) }}</n-text>
        </div>
      </template>
    </info-card>
  </div>
</template>

<style scoped>
.tip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.subtitle {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 4px;
}
.dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: orange;
  margin-right: 5px;
}
</style>
