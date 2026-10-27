import { computed, ref } from 'vue'
import { createI18n } from 'vue-i18n'
import { enUS, jaJP, dateEnUS, dateJaJP } from 'naive-ui'
import en from './en'
import ja from './ja'
import { GetLocale, SetLocale } from '@go/app/app'

export type Locale = 'en' | 'ja'
/** LocaleMode is what the user picks: a language or "follow the system". */
export type LocaleMode = Locale | 'system'

export const i18n = createI18n({
  legacy: false,
  locale: 'en',
  fallbackLocale: 'en',
  messages: { en, ja },
})

function systemLocale(): Locale {
  return navigator.language.toLowerCase().startsWith('ja') ? 'ja' : 'en'
}

const mode = ref<LocaleMode>('system')

function apply() {
  i18n.global.locale.value = mode.value === 'system' ? systemLocale() : mode.value
}
apply()

GetLocale().then((saved) => {
  if (saved === 'en' || saved === 'ja') mode.value = saved
  apply()
})

/** Naive UI's own component strings (date pickers, empty states, ...). */
const naiveLocales = {
  en: { locale: enUS, dateLocale: dateEnUS },
  ja: { locale: jaJP, dateLocale: dateJaJP },
}

/**
 * Translations for strings that originate in Go (mutation names,
 * descriptions and control labels). Unknown keys return the input unchanged.
 */
export function useMutationText() {
  function lookup(section: 'names' | 'descriptions' | 'components', key: string): string {
    const msgs = i18n.global.getLocaleMessage(i18n.global.locale.value) as any
    return msgs?.mutations?.[section]?.[key] ?? key
  }
  return {
    mutationName: (name: string) => lookup('names', name),
    mutationDescription: (name: string, fallback: string) => {
      const v = lookup('descriptions', name)
      return v === name ? fallback : v
    },
    componentName: (name: string) => lookup('components', name),
  }
}

export function useLocale() {
  const current = computed(() => i18n.global.locale.value as Locale)
  const naive = computed(() => naiveLocales[current.value])

  function setMode(m: LocaleMode) {
    mode.value = m
    apply()
    void SetLocale(m === 'system' ? '' : m)
  }

  return { mode, current, naive, setMode }
}
