import { createRouter, createWebHashHistory } from 'vue-router'
import HomeView from './views/HomeView.vue'
import OutputView from './views/OutputView.vue'
import ModulesView from './views/ModulesView.vue'
import MutatorView from './views/MutatorView.vue'
import SettingsView from './views/SettingsView.vue'

// meta.title is an i18n key under "nav".
export const routes = [
  { path: '/', name: 'home', component: HomeView, meta: { title: 'nav.home' } },
  { path: '/output', name: 'output', component: OutputView, meta: { title: 'nav.output' } },
  { path: '/modules', name: 'modules', component: ModulesView, meta: { title: 'nav.modules' } },
  { path: '/tracking', name: 'tracking', component: MutatorView, meta: { title: 'nav.tracking' } },
  { path: '/settings', name: 'settings', component: SettingsView, meta: { title: 'nav.settings' } },
]

export const router = createRouter({
  history: createWebHashHistory(),
  routes,
})
