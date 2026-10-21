import { computed, ref, watchEffect } from 'vue'
import { darkTheme } from 'naive-ui'
import {
  GetSeedColor, GetSystemAccentColor, GetTheme, GetUseSystemAccent, SetSeedColor, SetTheme, SetUseSystemAccent,
} from '../../wailsjs/go/app/App'
import { WindowSetBackgroundColour } from '../../wailsjs/runtime/runtime'
import { cssVariables, generatePalette, themeOverrides as md3Overrides } from '../theme/md3'

export type ThemeMode = 'light' | 'dark' | 'system'

export const defaultSeed = '#EC4899'

const mode = ref<ThemeMode>('dark')
const seed = ref(defaultSeed)
const useSystemAccent = ref(false)
const systemAccent = ref('')
const systemDark = ref(window.matchMedia('(prefers-color-scheme: dark)').matches)
window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => (systemDark.value = e.matches))

GetTheme().then((t) => {
  if (t === 'light' || t === 'dark' || t === 'system') mode.value = t
})
GetSeedColor().then((c) => {
  if (/^#[0-9a-f]{6}$/i.test(c)) seed.value = c
})
GetUseSystemAccent().then((v) => {
  useSystemAccent.value = v
  if (v) void refreshSystemAccent()
})

async function refreshSystemAccent() {
  try {
    systemAccent.value = await GetSystemAccentColor()
  } catch {
    systemAccent.value = ''
  }
}

// Windows has no change notification we can reach from here, so pick up a
// changed accent colour whenever the window regains focus.
window.addEventListener('focus', () => {
  if (useSystemAccent.value) void refreshSystemAccent()
})

// The system accent wins when enabled and readable; otherwise the user's seed.
const effectiveSeed = computed(() => (useSystemAccent.value && systemAccent.value ? systemAccent.value : seed.value))

const isDark = computed(() => (mode.value === 'system' ? systemDark.value : mode.value === 'dark'))
const palette = computed(() => generatePalette(effectiveSeed.value, isDark.value))
const theme = computed(() => (isDark.value ? darkTheme : null))
const themeOverrides = computed(() => md3Overrides(palette.value))

// Expose the scheme as --md-sys-color-* for scoped CSS, and keep the native
// window background in step so it never shows the wrong colour on resize.
watchEffect(() => {
  const root = document.documentElement
  for (const [name, value] of Object.entries(cssVariables(palette.value))) root.style.setProperty(name, value)
  root.style.colorScheme = isDark.value ? 'dark' : 'light'
  const s = palette.value.surfaceContainer
  try {
    WindowSetBackgroundColour((s >> 16) & 255, (s >> 8) & 255, s & 255, 255)
  } catch {
    // runtime unavailable (e.g. plain browser dev) — ignore
  }
})

export function useTheme() {
  function setMode(m: ThemeMode) {
    mode.value = m
    void SetTheme(m)
  }

  // setSeed previews a colour; saveSeed persists the current one. They are
  // separate so dragging in the colour picker does not write the settings
  // file on every move.
  function setSeed(c: string) {
    seed.value = c
  }

  function saveSeed() {
    void SetSeedColor(seed.value)
  }

  function setUseSystemAccent(v: boolean) {
    useSystemAccent.value = v
    void SetUseSystemAccent(v)
    if (v) void refreshSystemAccent()
  }

  return {
    mode, seed, useSystemAccent, systemAccent, isDark, palette, theme, themeOverrides,
    setMode, setSeed, saveSeed, setUseSystemAccent,
  }
}
