import { onBeforeUnmount, ref, type Ref } from 'vue'
import { Events as WailsEvents } from '@wailsio/runtime'

/**
 * Subscribes to a Wails event for the lifetime of the component.
 */
export function useEvent<T>(name: string, handler: (data: T) => void): void {
  const off = WailsEvents.On(name, (ev) => handler(ev.data as T))
  onBeforeUnmount(off)
}

/**
 * Loads a value from the backend once and keeps it updated from an event.
 */
export function useBackendState<T>(load: () => Promise<T>, event: string, initial: T): Ref<T> {
  const state = ref(initial) as Ref<T>
  load().then((v) => (state.value = v))
  useEvent<T>(event, (v) => (state.value = v))
  return state
}

export const Events = {
  log: 'log:line',
  modules: 'modules:changed',
  avatar: 'avatar:changed',
  oscStats: 'osc:stats',
  parameters: 'parameters:changed',
} as const
