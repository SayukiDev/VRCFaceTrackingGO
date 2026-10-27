<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NCard, NSelect, NText, useMessage, type SelectOption } from 'naive-ui'
import { GetModuleCatalog, GetTrackingAssignment, SetTrackingAssignment } from '@go/app/app'
import type * as router from '@go/internal/router'
import { Events, useEvent } from '../composables/useBackend'

// Manual choice of the eye and face module. It is not saved: the next
// launch starts from the priority list on the Modules page again.
const { t } = useI18n()
const message = useMessage()
const catalog = ref<router.Registered[]>([])
const eye = ref('')
const face = ref('')
// applying: a re-initialisation is in flight (cleared by the next module
// list update).
const applying = ref(false)

async function refresh() {
  const [c, a] = await Promise.all([GetModuleCatalog(), GetTrackingAssignment()])
  catalog.value = c
  eye.value = a.eye
  face.value = a.expression
}
refresh()
useEvent(Events.modules, () => {
  applying.value = false
  refresh()
})

function options(supports: (m: router.Registered) => boolean): SelectOption[] {
  return [
    { label: t('home.assignAuto'), value: '' },
    ...catalog.value.filter(supports).map((m) => ({ label: m.name, value: m.id })),
  ]
}
const eyeOptions = computed(() => options((m) => m.supportsEye))
const faceOptions = computed(() => options((m) => m.supportsExpression))

async function apply(newEye: string, newFace: string) {
  const [oldEye, oldFace] = [eye.value, face.value]
  eye.value = newEye
  face.value = newFace
  applying.value = true
  try {
    await SetTrackingAssignment(newEye, newFace)
  } catch (e) {
    applying.value = false
    eye.value = oldEye
    face.value = oldFace
    message.error(String(e))
  }
}
</script>

<template>
  <n-card size="small">
    <div class="row">
      <div class="info">
        <div class="subtitle">{{ t('home.assignTitle') }}</div>
        <n-text depth="3">{{ t('home.assignBody') }}</n-text>
      </div>
      <div class="field">
        <n-text depth="3">{{ t('modules.eye') }}</n-text>
        <n-select :value="eye" :options="eyeOptions" :loading="applying" :disabled="applying"
          @update:value="(v: string) => apply(v, face)" />
      </div>
      <div class="field">
        <n-text depth="3">{{ t('modules.face') }}</n-text>
        <n-select :value="face" :options="faceOptions" :loading="applying" :disabled="applying"
          @update:value="(v: string) => apply(eye, v)" />
      </div>
    </div>
  </n-card>
</template>

<style scoped>
.row {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 4px 8px;
}
.info {
  flex: 1;
}
.subtitle {
  font-size: var(--md-sys-typescale-title-medium-size);
  line-height: var(--md-sys-typescale-title-medium-line-height);
  font-weight: var(--md-sys-typescale-title-medium-weight);
  letter-spacing: var(--md-sys-typescale-title-medium-tracking);
  margin-bottom: 4px;
}
.field {
  width: 200px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
</style>
