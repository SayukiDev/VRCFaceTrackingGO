import { computed, ref } from 'vue'
import { Environment } from '../../wailsjs/runtime/runtime'

// platform is the backend's GOOS ("windows", "linux", "darwin"). VRChat and
// avatar detection and the system accent colour only exist on Windows.
const platform = ref('windows')
try {
  Environment()
    .then((e) => (platform.value = e.platform))
    .catch(() => {})
} catch {
  // runtime unavailable (e.g. plain browser dev) — assume Windows
}

const isWindows = computed(() => platform.value === 'windows')

export function usePlatform() {
  return { platform, isWindows }
}
