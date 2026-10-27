import { computed, ref } from 'vue'
import { System } from '@wailsio/runtime'

// platform is the backend's GOOS ("windows", "linux", "darwin"). Only the
// system accent colour is Windows-specific.
const platform = ref('windows')
try {
  System.Environment()
    .then((e) => (platform.value = e.OS))
    .catch(() => {})
} catch {
  // runtime unavailable (e.g. plain browser dev) — assume Windows
}

const isWindows = computed(() => platform.value === 'windows')

export function usePlatform() {
  return { platform, isWindows }
}
