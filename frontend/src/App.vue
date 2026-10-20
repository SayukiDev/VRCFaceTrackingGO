<script setup lang="ts">
import { computed, h, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  NConfigProvider, NGlobalStyle, NLayout, NLayoutSider, NLayoutContent, NMenu, NIcon, NMessageProvider, NDialogProvider,
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

function link(to: string, key: string) {
  return () => h(RouterLink, { to }, { default: () => t(key) })
}

const menuOptions = computed<MenuOption[]>(() => [
  { key: 'home', label: link('/', 'nav.home'), icon: renderIcon(Home24Regular) },
  { key: 'output', label: link('/output', 'nav.output'), icon: renderIcon(TextBulletListLtr24Regular) },
  { key: 'modules', label: link('/modules', 'nav.modules'), icon: renderIcon(Apps24Regular) },
  { key: 'tracking', label: link('/tracking', 'nav.tracking'), icon: renderIcon(Options24Regular) },
])

const bottomOptions = computed<MenuOption[]>(() => [
  { key: 'settings', label: link('/settings', 'nav.settings'), icon: renderIcon(Settings24Regular) },
])

const activeKey = computed(() => String(route.name ?? 'home'))
const title = computed(() => t(String(route.meta.title ?? '')))
</script>

<template>
  <n-config-provider :theme="theme" :theme-overrides="themeOverrides" :locale="naive.locale" :date-locale="naive.dateLocale">
    <n-global-style />
    <n-message-provider>
      <n-dialog-provider>
        <n-layout has-sider class="shell">
          <n-layout-sider
            bordered
            collapse-mode="width"
            :collapsed-width="64"
            :width="220"
            :collapsed="collapsed"
            show-trigger
            @collapse="collapsed = true"
            @expand="collapsed = false"
          >
            <div class="sider">
              <div class="app-title" style="--wails-draggable: drag">
                <img src="./assets/icon.png" alt="" class="app-icon" />
                <span v-if="!collapsed">VRCFaceTrackingGO</span>
              </div>
              <n-menu :collapsed="collapsed" :collapsed-width="64" :collapsed-icon-size="22" :options="menuOptions" :value="activeKey" />
              <div class="spacer" />
              <n-menu :collapsed="collapsed" :collapsed-width="64" :collapsed-icon-size="22" :options="bottomOptions" :value="activeKey" />
            </div>
          </n-layout-sider>
          <div class="main">
            <TitleBar />
            <n-layout-content content-style="padding: 4px 32px 24px;">
              <h1 class="page-title">{{ title }}</h1>
              <router-view />
            </n-layout-content>
          </div>
        </n-layout>
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<style scoped>
.shell {
  height: 100vh;
}
.main {
  display: flex;
  flex-direction: column;
  height: 100%;
  flex: 1;
  min-width: 0;
}
.app-title {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 16px 22px 8px;
  font-size: 12px;
  opacity: 0.8;
}
.app-icon {
  width: 20px;
  height: 20px;
  flex: none;
}
.sider {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.spacer {
  flex: 1;
}
.page-title {
  font-size: 28px;
  font-weight: 600;
  margin: 0 0 12px;
}
</style>
