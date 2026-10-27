<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton, NColorPicker, NCollapse, NCollapseItem, NInput, NInputNumber, NSelect, NSpace, NSwitch, NText, useMessage,
} from 'naive-ui'
import { Checkmark20Regular, ArrowRouting24Regular, Color24Regular, PaintBrush24Regular, LocalLanguage24Regular, PlugConnected24Regular, Send24Regular, Timer24Regular, Warning24Regular, Info24Regular } from '@vicons/fluent'
import {
  GetOscTarget, SetOscTarget, GetOscMode, SetOscMode, GetAllParametersRelevant, SetAllParametersRelevant, ForceReInit, ResetVRCFT,
  ResetAvatarConfigs, GetMetadata, OpenURL, GetModuleTiming, SetModuleTiming, GetOutputMode, SetOutputMode,
} from '@go/app/app'
import * as metadata from '@go/internal/metadata'
import * as osc from '@go/internal/osc'
import * as settings from '@go/internal/settings'
import { defaultSeed, useTheme, type ThemeMode } from '../composables/useTheme'
import { useLocale, type LocaleMode } from '../i18n'
import { usePlatform } from '../composables/usePlatform'
import InfoCard from '../components/InfoCard.vue'

const { t } = useI18n()
const message = useMessage()

const { mode, setMode, seed, setSeed, saveSeed, useSystemAccent, setUseSystemAccent } = useTheme()
// The system accent colour and VRChat / avatar detection only exist on Windows.
const { isWindows } = usePlatform()
// M3 baseline purple, a few hues around the wheel, and the default app pink.
const seedSwatches = ['#6750A4', '#0061A4', '#006A6A', '#006E1C', '#6A5F00', '#9C4146', '#8B5000', defaultSeed]

function pickSwatch(c: string) {
  setSeed(c)
  saveSeed()
}
const themeOptions = computed(() => [
  { label: t('settings.themeLight'), value: 'light' },
  { label: t('settings.themeDark'), value: 'dark' },
  { label: t('settings.themeSystem'), value: 'system' },
])

const { mode: localeMode, setMode: setLocaleMode } = useLocale()
// Language names are shown in their own language on purpose.
const localeOptions = computed(() => [
  { label: 'English', value: 'en' },
  { label: '日本語', value: 'ja' },
  { label: t('settings.languageSystem'), value: 'system' },
])

const outputMode = ref('parameters')
GetOutputMode().then((v) => (outputMode.value = v))
const outputOptions = computed(() => [
  { label: t('settings.outputParameters'), value: 'parameters' },
  { label: t('settings.outputNative'), value: 'native' },
  { label: t('settings.outputBoth'), value: 'both' },
])

async function applyOutputMode(v: string) {
  try {
    await SetOutputMode(v)
    outputMode.value = v
  } catch (e) {
    message.error(String(e))
  }
}

const oscMode = ref('oscquery')
GetOscMode().then((v) => (oscMode.value = v))
const oscModeOptions = computed(() => [
  { label: t('settings.oscModeQuery'), value: 'oscquery' },
  { label: t('settings.oscModeOsc'), value: 'osc' },
])

async function applyOscMode(v: string) {
  try {
    await SetOscMode(v)
    oscMode.value = v
  } catch (e) {
    message.error(String(e))
  }
}

const target = ref<osc.Target>(new osc.Target({ inPort: 9001, outPort: 9000, address: '127.0.0.1' }))
GetOscTarget().then((v) => (target.value = v))

async function applyTarget() {
  try {
    await SetOscTarget(target.value)
  } catch (e) {
    message.error(String(e))
    target.value = await GetOscTarget()
  }
}

const timing = ref<settings.Timing>(new settings.Timing({ updateIntervalMs: 10, initTimeoutSec: 15, sendIntervalMs: 10 }))
GetModuleTiming().then((v) => (timing.value = v))

async function applyTiming() {
  try {
    await SetModuleTiming(timing.value)
  } catch (e) {
    message.error(String(e))
    timing.value = await GetModuleTiming()
  }
}

const gremlin = ref(false)
const allRelevant = ref(false)
GetAllParametersRelevant().then((v) => (allRelevant.value = v))

function setGremlin(v: boolean) {
  gremlin.value = v
  if (!v && allRelevant.value) setAllRelevant(false)
}

function setAllRelevant(v: boolean) {
  allRelevant.value = v
  void SetAllParametersRelevant(v)
}

async function resetVRCFT() {
  await ResetVRCFT()
  message.info(t('settings.resetVRCFTDone'))
}

async function resetAvatarConfigs() {
  try {
    await ResetAvatarConfigs()
    message.success(t('settings.resetAvatarDone'))
  } catch (e) {
    message.error(String(e))
  }
}

const meta = ref<metadata.Metadata>(new metadata.Metadata({ name: 'VRCFaceTrackingGO' }))
GetMetadata().then((v) => (meta.value = v))
</script>

<template>
  <div class="stack">
    <info-card :icon="Color24Regular">
      <div class="header">{{ t('settings.theme') }}</div>
      <n-text depth="3">{{ t('settings.themeDesc') }}</n-text>
      <template #aside>
        <n-select :value="mode" :options="themeOptions" style="width: 200px" @update:value="(v: ThemeMode) => setMode(v)" />
      </template>
    </info-card>

    <info-card :icon="PaintBrush24Regular">
      <div class="header">{{ t('settings.seedColor') }}</div>
      <n-text depth="3">{{ t('settings.seedColorDesc') }}</n-text>
      <label v-if="isWindows" class="accent-switch">
        <n-switch :value="useSystemAccent" @update:value="setUseSystemAccent" />
        <span>{{ t('settings.useSystemAccent') }}</span>
      </label>
      <div class="swatches" :class="{ disabled: isWindows && useSystemAccent }">
        <button
          v-for="c in seedSwatches"
          :key="c"
          class="swatch"
          :class="{ selected: c.toUpperCase() === seed.toUpperCase() }"
          :style="{ background: c }"
          :title="c"
          :disabled="isWindows && useSystemAccent"
          @click="pickSwatch(c)"
        >
          <n-icon v-if="c.toUpperCase() === seed.toUpperCase()" :size="20"><checkmark20-regular /></n-icon>
        </button>
      </div>
      <div class="custom-color">
        <span class="custom-label">{{ t('settings.customColor') }}</span>
        <n-color-picker
          :value="seed"
          :modes="['hex']"
          :show-alpha="false"
          :disabled="isWindows && useSystemAccent"
          @update:value="(v: string) => setSeed(v.toUpperCase())"
          @update:show="(open: boolean) => !open && saveSeed()"
        />
      </div>
    </info-card>

    <info-card :icon="LocalLanguage24Regular">
      <div class="header">{{ t('settings.language') }}</div>
      <n-text depth="3">{{ t('settings.languageDesc') }}</n-text>
      <template #aside>
        <n-select :value="localeMode" :options="localeOptions" style="width: 200px" @update:value="(v: LocaleMode) => setLocaleMode(v)" />
      </template>
    </info-card>

    <info-card :icon="ArrowRouting24Regular">
      <div class="header">{{ t('settings.output') }}</div>
      <n-text depth="3">{{ t('settings.outputDesc') }}</n-text>
      <template #aside>
        <n-select :value="outputMode" :options="outputOptions" style="width: 260px" @update:value="applyOutputMode" />
      </template>
    </info-card>

    <info-card :icon="PlugConnected24Regular">
      <div class="header">{{ t('settings.oscMode') }}</div>
      <n-text depth="3">{{ t('settings.oscModeDesc') }}</n-text>
      <template #aside>
        <n-select :value="oscMode" :options="oscModeOptions" style="width: 260px" @update:value="applyOscMode" />
      </template>
    </info-card>

    <info-card :icon="PlugConnected24Regular">
      <div class="header">{{ t('settings.osc') }}</div>
      <n-text depth="3">{{ t('settings.oscDesc') }}</n-text>
      <template #aside>
        <n-space align="end">
          <div>
            <div class="field-label">{{ t('settings.inPort') }}</div>
            <n-input-number v-model:value="target.inPort" :min="1" :max="65535" :show-button="true" @blur="applyTarget" />
          </div>
          <div>
            <div class="field-label">{{ t('settings.ipAddress') }}</div>
            <n-input v-model:value="target.address" placeholder="127.0.0.1" @blur="applyTarget" />
          </div>
          <div>
            <div class="field-label">{{ t('settings.outPort') }}</div>
            <n-input-number v-model:value="target.outPort" :min="1" :max="65535" :show-button="true" @blur="applyTarget" />
          </div>
        </n-space>
      </template>
    </info-card>

    <info-card :icon="Timer24Regular">
      <div class="header">{{ t('settings.timing') }}</div>
      <n-text depth="3">{{ t('settings.timingDesc') }}</n-text>
      <template #aside>
        <n-space align="end">
          <div>
            <div class="field-label">{{ t('settings.updateInterval') }}</div>
            <n-input-number v-model:value="timing.updateIntervalMs" :min="1" :max="1000" :step="1" :show-button="true" style="width: 140px" @blur="applyTiming">
              <template #suffix>ms</template>
            </n-input-number>
          </div>
          <div>
            <div class="field-label">{{ t('settings.initTimeout') }}</div>
            <n-input-number v-model:value="timing.initTimeoutSec" :min="1" :max="300" :step="1" :show-button="true" style="width: 140px" @blur="applyTiming">
              <template #suffix>s</template>
            </n-input-number>
          </div>
        </n-space>
      </template>
    </info-card>

    <info-card :icon="Send24Regular">
      <div class="header">{{ t('settings.sendTiming') }}</div>
      <n-text depth="3">{{ t('settings.sendTimingDesc') }}</n-text>
      <template #aside>
        <div>
          <div class="field-label">{{ t('settings.sendInterval') }}</div>
          <n-input-number v-model:value="timing.sendIntervalMs" :min="1" :max="1000" :step="1" :show-button="true" style="width: 140px" @blur="applyTiming">
            <template #suffix>ms</template>
          </n-input-number>
        </div>
      </template>
    </info-card>

    <n-collapse arrow-placement="right">
      <n-collapse-item name="risky">
        <template #header>
          <info-card :icon="Warning24Regular" class="flat">
            <div class="header">{{ t('settings.risky') }}</div>
            <n-text depth="3">{{ t('settings.riskyDesc') }}</n-text>
            <template #aside>
              <div @click.stop>
                <n-switch :value="gremlin" @update:value="setGremlin">
                  <template #checked>{{ t('settings.gremlin') }}</template>
                </n-switch>
              </div>
            </template>
          </info-card>
        </template>
        <div class="stack">
          <info-card>
            <div class="header">{{ t('settings.forceRelevancy') }}</div>
            <n-text depth="3">{{ t('settings.forceRelevancyDesc') }}</n-text>
            <template #aside><n-switch :value="allRelevant" :disabled="!gremlin" @update:value="setAllRelevant" /></template>
          </info-card>
          <info-card>
            <div class="header">{{ t('settings.forceReInit') }}</div>
            <n-text depth="3">{{ t('settings.forceReInitDesc') }}</n-text>
            <template #aside><n-button :disabled="!gremlin" @click="ForceReInit()">{{ t('settings.reInit') }}</n-button></template>
          </info-card>
          <info-card>
            <div class="header">{{ t('settings.resetVRCFT') }}</div>
            <n-text depth="3">{{ t('settings.resetVRCFTDesc') }}</n-text>
            <template #aside><n-button :disabled="!gremlin" @click="resetVRCFT">{{ t('settings.reset') }}</n-button></template>
          </info-card>
          <info-card v-if="isWindows">
            <div class="header">{{ t('settings.resetAvatar') }}</div>
            <n-text depth="3">{{ t('settings.resetAvatarDesc') }}</n-text>
            <template #aside><n-button :disabled="!gremlin" @click="resetAvatarConfigs">{{ t('settings.reset') }}</n-button></template>
          </info-card>
        </div>
      </n-collapse-item>
    </n-collapse>

    <div class="section">{{ t('settings.about') }}</div>
    <info-card :icon="Info24Regular">
      <div class="header">{{ meta.name }}</div>
      <n-text depth="3">{{ meta.info.copyright }}</n-text>
      <div class="links">
        <n-button text type="primary" @click="OpenURL(meta.info.noticesUrl)">{{ t('settings.notices') }}</n-button>
        <n-button text type="primary" @click="OpenURL(meta.info.issuesUrl)">{{ t('settings.bugReport') }}</n-button>
      </div>
      <n-text depth="3" class="clone">git clone {{ meta.info.repository }}</n-text>
      <n-text depth="3" class="disclaimer">{{ t('settings.disclaimer') }}</n-text>
      <template #aside><n-text depth="3">{{ meta.info.productVersion }}</n-text></template>
    </info-card>
  </div>
</template>

<style scoped>
.header {
  font-size: var(--md-sys-typescale-title-small-size);
  line-height: var(--md-sys-typescale-title-small-line-height);
  font-weight: var(--md-sys-typescale-title-small-weight);
  letter-spacing: var(--md-sys-typescale-title-small-tracking);
}
.accent-switch {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-top: 16px;
  cursor: pointer;
}
.swatches {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 16px;
}
.swatches.disabled {
  opacity: 0.38;
}
.swatch {
  width: 40px;
  height: 40px;
  padding: 0;
  border: none;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  cursor: pointer;
  /* Inner ring keeps light swatches visible on the card. */
  box-shadow: inset 0 0 0 1px var(--md-sys-color-outline-variant);
  transition: box-shadow 0.2s cubic-bezier(0.2, 0, 0, 1);
}
.swatch:disabled {
  cursor: default;
}
.swatch:not(:disabled):hover {
  box-shadow: inset 0 0 0 1px var(--md-sys-color-outline-variant), 0 0 0 4px var(--md-sys-color-on-surface-hover);
}
.swatch.selected {
  box-shadow: 0 0 0 2px var(--md-sys-color-surface-container-low), 0 0 0 4px var(--md-sys-color-on-surface);
}
.custom-color {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-top: 16px;
}
.custom-label {
  flex: none;
}
.custom-color :deep(.n-color-picker) {
  flex: 1;
}
.field-label {
  font-size: 12px;
  opacity: 0.8;
  margin-bottom: 4px;
}
.section {
  font-size: var(--md-sys-typescale-title-medium-size);
  line-height: var(--md-sys-typescale-title-medium-line-height);
  font-weight: var(--md-sys-typescale-title-medium-weight);
  letter-spacing: var(--md-sys-typescale-title-medium-tracking);
  margin: 28px 0 4px 1px;
}
.links {
  display: flex;
  gap: 16px;
  margin: 8px 0 4px;
}
.clone,
.disclaimer {
  display: block;
  font-size: 12px;
  user-select: text;
}
.disclaimer {
  margin-top: 8px;
}
.flat :deep(.n-card) {
  border: none;
  background: transparent;
}
</style>
