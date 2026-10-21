<script setup lang="ts">
import { computed, h, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  NConfigProvider, NGlobalStyle, NLayout, NLayoutSider, NMenu, NIcon, NMessageProvider, NDialogProvider,
  type MenuOption,
} from 'naive-ui'
import { Home24Regular, TextBulletListLtr24Regular, Apps24Regular, Options24Regular, Settings24Regular } from '@vicons/fluent'
import { useTheme } from './composables/useTheme'
import { useLocale } from './i18n'
import TitleBar from './components/TitleBar.vue'

const { t } = useI18n()
const { theme, themeOverrides } = useTheme()
const { naive } = useLocale()
const route = useRoute()
const collapsed = ref(false)

function renderIcon(icon: any) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

function link(to: string, label: () => string) {
  return () => h(RouterLink, { to }, { default: label })
}

const menuOptions = computed<MenuOption[]>(() => [
  { key: 'home', label: link('/', () => t('nav.home')), icon: renderIcon(Home24Regular) },
  { key: 'output', label: link('/output', () => t('nav.output')), icon: renderIcon(TextBulletListLtr24Regular) },
  { key: 'modules', label: link('/modules', () => t('nav.modules')), icon: renderIcon(Apps24Regular) },
  { key: 'tracking', label: link('/tracking', () => t('nav.tracking')), icon: renderIcon(Options24Regular) },
])

const bottomOptions = computed<MenuOption[]>(() => [
  { key: 'settings', label: link('/settings', () => t('nav.settings')), icon: renderIcon(Settings24Regular) },
])

const activeKey = computed(() => String(route.name ?? 'home'))
const title = computed(() => t(String(route.meta.title ?? '')))
</script>

<template>
  <n-config-provider :theme="theme" :theme-overrides="themeOverrides" :locale="naive.locale" :date-locale="naive.dateLocale">
    <n-global-style />
    <n-message-provider>
      <n-dialog-provider>
        <div class="shell">
          <TitleBar />
          <n-layout has-sider class="body">
            <n-layout-sider
              collapse-mode="width"
              :collapsed-width="72"
              :width="200"
              :collapsed="collapsed"
              show-trigger
              @collapse="collapsed = true"
              @expand="collapsed = false"
            >
              <div class="sider">
                <n-menu :collapsed="collapsed" :collapsed-width="72" :collapsed-icon-size="20" :icon-size="20" :root-indent="24" :options="menuOptions" :value="activeKey" />
                <div class="spacer" />
                <n-menu :collapsed="collapsed" :collapsed-width="72" :collapsed-icon-size="20" :icon-size="20" :root-indent="24" :options="bottomOptions" :value="activeKey" />
              </div>
            </n-layout-sider>
            <main class="panel">
              <header class="page-header">
                <h1 class="page-title">{{ title }}</h1>
                <!-- Views teleport their page level actions here. -->
                <div id="page-actions" class="page-actions" />
              </header>
              <div class="page">
                <router-view />
              </div>
            </main>
          </n-layout>
        </div>
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<style scoped>
.shell {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: var(--md-sys-color-surface-container);
}
.body {
  flex: 1;
  min-height: 0;
}
.sider {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 4px 0 12px;
  box-sizing: border-box;
}
/* M3 navigation drawer: the active indicator is inset 12dp from both edges. */
.sider :deep(.n-menu-item-content::before) {
  left: 12px;
  right: 12px;
}
.sider :deep(.n-menu-item-content-header) {
  font-weight: var(--md-sys-typescale-label-large-weight);
  letter-spacing: var(--md-sys-typescale-label-large-tracking);
}
.spacer {
  flex: 1;
}
/* The content pane sits on the window surface with a large top-start corner. */
.panel {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: var(--md-sys-color-surface);
  border-top: 1px solid var(--md-sys-color-outline-variant);
  border-left: 1px solid var(--md-sys-color-outline-variant);
  border-top-left-radius: 16px;
}
.page-header {
  flex: none;
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 32px 40px 16px;
}
.page-title {
  flex: 1;
  margin: 0;
  color: var(--md-sys-color-on-surface);
  font-size: var(--md-sys-typescale-headline-medium-size);
  line-height: var(--md-sys-typescale-headline-medium-line-height);
  font-weight: var(--md-sys-typescale-headline-medium-weight);
  letter-spacing: var(--md-sys-typescale-headline-medium-tracking);
}
.page-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.page {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 8px 40px 24px;
}
</style>
