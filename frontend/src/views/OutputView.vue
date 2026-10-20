<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NButtonGroup, NIcon, NScrollbar, NSpace, NText, NTooltip, type ScrollbarInst } from 'naive-ui'
import { Copy24Regular, Save24Regular } from '@vicons/fluent'
import { GetLogs, CopyLogs, SaveLogs } from '../../wailsjs/go/app/App'
import type { logging } from '../../wailsjs/go/models'
import { Events, useEvent } from '../composables/useBackend'

const { t } = useI18n()
const lines = ref<logging.Line[]>([])
const status = ref('')
const scroller = ref<ScrollbarInst | null>(null)

// Debug lines are kept for copy/save but hidden, like the original Output page.
const visible = computed(() => lines.value.filter((l) => l.level !== 'DEBUG'))

function scrollToBottom() {
  nextTick(() => scroller.value?.scrollTo({ top: Number.MAX_SAFE_INTEGER }))
}

GetLogs().then((l) => {
  lines.value = l
  scrollToBottom()
})
useEvent<logging.Line>(Events.log, (l) => {
  lines.value.push(l)
  scrollToBottom()
})

const levelColors: Record<string, string> = {
  WARN: '#FFFF23',
  ERROR: '#DB2543',
  FATAL: '#A70023',
  DPANIC: '#A70023',
  PANIC: '#A70023',
}

async function copy() {
  await CopyLogs()
  status.value = t('output.copied')
}

async function save() {
  const name = await SaveLogs()
  status.value = name ? t('output.saved', { name }) : t('output.cancelled')
}
</script>

<template>
  <div class="output">
    <div class="toolbar">
      <n-space align="center">
        <n-button-group>
          <n-tooltip>
            <template #trigger>
              <n-button @click="copy"><template #icon><n-icon><copy24-regular /></n-icon></template></n-button>
            </template>
            {{ t('output.copy') }}
          </n-tooltip>
          <n-tooltip>
            <template #trigger>
              <n-button @click="save"><template #icon><n-icon><save24-regular /></n-icon></template></n-button>
            </template>
            {{ t('output.save') }}
          </n-tooltip>
        </n-button-group>
        <n-text depth="3">{{ status }}</n-text>
      </n-space>
    </div>
    <div class="log">
      <n-scrollbar ref="scroller">
        <pre class="lines"><div v-for="(l, i) in visible" :key="i" :style="{ color: levelColors[l.level] }">{{ l.message }}</div></pre>
      </n-scrollbar>
    </div>
  </div>
</template>

<style scoped>
.output {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 110px);
  gap: 16px;
}
.toolbar {
  padding: 8px;
  border-radius: 6px;
  background: rgba(127, 127, 127, 0.08);
}
.log {
  flex: 1;
  min-height: 0;
  border-radius: 6px;
  background: rgba(127, 127, 127, 0.08);
  padding: 10px;
  user-select: text;
}
.lines {
  margin: 0;
  font: 13px/1.5 Consolas, "Cascadia Mono", monospace;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
