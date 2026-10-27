<script setup lang="ts">
import { h, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NCheckbox, NIcon, useDialog } from 'naive-ui'
import { Subtract24Regular, Maximize24Regular, SquareMultiple24Regular, Dismiss24Regular } from '@vicons/fluent'
import { Window } from '@wailsio/runtime'
import { CloseAllWindows, GetHideTrayNotice, SetHideTrayNotice } from '@go/app/app'

const { t } = useI18n()
const dialog = useDialog()
const maximised = ref(false)

// close tells the user that the application keeps running in the tray,
// unless they opted out, and then closes every window.
async function close() {
  if (await GetHideTrayNotice().catch(() => false)) {
    void CloseAllWindows()
    return
  }
  const hide = ref(false)
  dialog.info({
    title: t('titlebar.trayNotice.title'),
    content: () =>
      h('div', [
        h('p', t('titlebar.trayNotice.body')),
        h(
          NCheckbox,
          { checked: hide.value, 'onUpdate:checked': (v: boolean) => (hide.value = v) },
          () => t('titlebar.trayNotice.dontShowAgain'),
        ),
      ]),
    positiveText: t('titlebar.trayNotice.ok'),
    negativeText: t('titlebar.trayNotice.cancel'),
    onPositiveClick: () => {
      if (hide.value) void SetHideTrayNotice(true)
      void CloseAllWindows()
    },
  })
}

async function refresh() {
  try {
    maximised.value = await Window.IsMaximised()
  } catch {
    // runtime unavailable (e.g. plain browser dev) — ignore
  }
}

function toggleMaximise() {
  // The runtime does not emit a maximise event; re-query after the change.
  Window.ToggleMaximise().then(refresh, () => {})
}

onMounted(() => {
  refresh()
  window.addEventListener('resize', refresh)
})
onUnmounted(() => window.removeEventListener('resize', refresh))
</script>

<template>
  <header class="titlebar" style="--wails-draggable: drag" @dblclick="toggleMaximise">
    <div class="brand">
      <img src="../assets/icon.png" alt="" class="app-icon" />
      <span>VRCFaceTrackingGO</span>
    </div>
    <div class="drag-area" />
    <div class="controls" style="--wails-draggable: no-drag" @dblclick.stop>
      <button class="ctl" :aria-label="$t('titlebar.minimise')" @click="Window.Minimise()">
        <n-icon :size="16"><Subtract24Regular /></n-icon>
      </button>
      <button class="ctl" :aria-label="maximised ? $t('titlebar.restore') : $t('titlebar.maximise')" @click="toggleMaximise">
        <n-icon :size="16">
          <SquareMultiple24Regular v-if="maximised" />
          <Maximize24Regular v-else />
        </n-icon>
      </button>
      <button class="ctl close" :aria-label="$t('titlebar.close')" @click="close">
        <n-icon :size="16"><Dismiss24Regular /></n-icon>
      </button>
    </div>
  </header>
</template>

<style scoped>
.titlebar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 40px;
  flex: none;
}
.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-left: 16px;
  color: var(--md-sys-color-on-surface-variant);
  font-size: var(--md-sys-typescale-body-small-size);
  line-height: var(--md-sys-typescale-body-small-line-height);
  letter-spacing: var(--md-sys-typescale-body-small-tracking);
  white-space: nowrap;
}
.app-icon {
  width: 16px;
  height: 16px;
}
.drag-area {
  flex: 1;
  height: 100%;
}
.controls {
  display: flex;
  height: 100%;
}
.ctl {
  width: 46px;
  height: 100%;
  border: 0;
  background: transparent;
  color: var(--md-sys-color-on-surface-variant);
  cursor: default;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: background 0.1s;
}
.ctl:hover {
  background: var(--md-sys-color-on-surface-hover);
}
.ctl.close:hover {
  background: #c42b1c;
  color: #fff;
}
</style>
