<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { NIcon } from 'naive-ui'
import { Subtract24Regular, Maximize24Regular, SquareMultiple24Regular, Dismiss24Regular } from '@vicons/fluent'
import { WindowMinimise, WindowToggleMaximise, WindowIsMaximised, Quit } from '../../wailsjs/runtime/runtime'

const maximised = ref(false)

async function refresh() {
  try {
    maximised.value = await WindowIsMaximised()
  } catch {
    // runtime unavailable (e.g. plain browser dev) — ignore
  }
}

function toggleMaximise() {
  WindowToggleMaximise()
  // The runtime does not emit a maximise event; re-query after the change.
  setTimeout(refresh, 50)
}

onMounted(() => {
  refresh()
  window.addEventListener('resize', refresh)
})
onUnmounted(() => window.removeEventListener('resize', refresh))
</script>

<template>
  <header class="titlebar" style="--wails-draggable: drag" @dblclick="toggleMaximise">
    <div class="drag-area" />
    <div class="controls" style="--wails-draggable: no-drag" @dblclick.stop>
      <button class="ctl" :title="$t('titlebar.minimise')" @click="WindowMinimise()">
        <n-icon :size="16"><Subtract24Regular /></n-icon>
      </button>
      <button class="ctl" :title="maximised ? $t('titlebar.restore') : $t('titlebar.maximise')" @click="toggleMaximise">
        <n-icon :size="16">
          <SquareMultiple24Regular v-if="maximised" />
          <Maximize24Regular v-else />
        </n-icon>
      </button>
      <button class="ctl close" :title="$t('titlebar.close')" @click="Quit()">
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
  height: 32px;
  flex: none;
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
  color: inherit;
  cursor: default;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: background 0.1s;
}
.ctl:hover {
  background: rgba(128, 128, 128, 0.2);
}
.ctl.close:hover {
  background: #c42b1c;
  color: #fff;
}
</style>
