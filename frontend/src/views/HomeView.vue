<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { NText, NAlert } from 'naive-ui'
import { PlugConnected24Regular, Person24Regular } from '@vicons/fluent'
import { GetModules, GetOscStatus, GetAvatar, GetParameterStats } from '../../wailsjs/go/app/App'
import type { manager, service, oscquery, sender } from '../../wailsjs/go/models'
import { Events, useBackendState } from '../composables/useBackend'
import InfoCard from '../components/InfoCard.vue'
import ModuleCard from '../components/ModuleCard.vue'

const { t } = useI18n()
const modules = useBackendState<manager.ModuleInfo[]>(GetModules, Events.modules, [])
const osc = useBackendState<service.OscStatus | null>(GetOscStatus, Events.oscStats, null)
const avatar = useBackendState<oscquery.AvatarInfo | null>(GetAvatar, Events.avatar, null)
const stats = useBackendState<sender.ParameterStats | null>(GetParameterStats, Events.parameters, null)
</script>

<template>
  <div class="stack">
    <module-card v-for="m in modules" :key="m.id" :module="m" />

    <n-alert v-if="modules.length === 0" :title="t('home.noModulesTitle')" type="warning">
      {{ t('home.noModulesBody') }}
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
