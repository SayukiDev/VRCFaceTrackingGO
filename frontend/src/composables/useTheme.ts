import { computed, ref } from 'vue'
import { darkTheme, type GlobalThemeOverrides } from 'naive-ui'
import { GetTheme, SetTheme } from '../../wailsjs/go/app/App'

export type ThemeMode = 'light' | 'dark' | 'system'

const mode = ref<ThemeMode>('dark')
const systemDark = ref(window.matchMedia('(prefers-color-scheme: dark)').matches)
window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => (systemDark.value = e.matches))

GetTheme().then((t) => {
  if (t === 'light' || t === 'dark' || t === 'system') mode.value = t
})

// Pink accent shared by both themes; only the primary colour family is
// overridden so Naive UI's own light/dark palettes stay intact.
const pink = {
  base: '#EC4899',
  hover: '#F472B6',
  pressed: '#DB2777',
  suppl: '#F9A8D4',
}

const lightOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: pink.base,
    primaryColorHover: pink.hover,
    primaryColorPressed: pink.pressed,
    primaryColorSuppl: pink.suppl,
  },
}

const darkOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: pink.hover,
    primaryColorHover: pink.suppl,
    primaryColorPressed: pink.base,
    primaryColorSuppl: pink.base,
  },
}

export function useTheme() {
  const isDark = computed(() => (mode.value === 'system' ? systemDark.value : mode.value === 'dark'))
  const theme = computed(() => (isDark.value ? darkTheme : null))
  const themeOverrides = computed(() => (isDark.value ? darkOverrides : lightOverrides))

  function setMode(m: ThemeMode) {
    mode.value = m
    void SetTheme(m)
  }

  return { mode, isDark, theme, themeOverrides, setMode }
}
