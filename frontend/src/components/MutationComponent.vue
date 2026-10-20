<script setup lang="ts">
import { NButton, NCheckbox, NSlider, NText } from 'naive-ui'
import type { mixer } from '../../wailsjs/go/models'
import { InvokeMutationButton, SetMutationRange, SetMutationValue } from '../../wailsjs/go/app/App'
import { useMutationText } from '../i18n'

const { componentName } = useMutationText()

const props = defineProps<{ mutation: string; component: mixer.ComponentState }>()

function setValue(v: unknown) {
  props.component.value = v
  void SetMutationValue(props.mutation, props.component.name, v)
}

function setRange(v: number | number[]) {
  if (!Array.isArray(v)) return
  props.component.value = { floor: v[0], ceil: v[1] }
  void SetMutationRange(props.mutation, props.component.name, v[0], v[1])
}

function invoke() {
  void InvokeMutationButton(props.mutation, props.component.name)
}

function format(v: number) {
  return Number(v).toFixed(2)
}
</script>

<template>
  <div class="component">
    <div class="label">{{ componentName(component.name) }}</div>

    <div v-if="component.kind === 'checkbox'" class="control right">
      <n-checkbox :checked="component.value as boolean" @update:checked="setValue" />
    </div>

    <div v-else-if="component.kind === 'slider'" class="control">
      <n-text class="value">{{ format(component.value as number) }}</n-text>
      <n-slider :value="component.value as number" :min="component.min" :max="component.max" :step="0.01" :tooltip="false" @update:value="setValue" />
    </div>

    <div v-else-if="component.kind === 'range'" class="control">
      <n-text class="value">{{ format((component.value as any).floor) }}</n-text>
      <n-slider
        range
        :value="[(component.value as any).floor, (component.value as any).ceil]"
        :min="component.min"
        :max="component.max"
        :step="0.01"
        :tooltip="false"
        @update:value="setRange"
      />
      <n-text class="value">{{ format((component.value as any).ceil) }}</n-text>
    </div>

    <div v-else-if="component.kind === 'button'" class="control right">
      <n-button size="small" @click="invoke">{{ componentName(component.name) }}</n-button>
    </div>
  </div>
</template>

<style scoped>
.component {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 10px 20px;
  margin: 1px 0;
  border-radius: 2px;
  background: rgba(127, 127, 127, 0.06);
}
.label {
  flex: 5;
}
.control {
  flex: 6;
  display: flex;
  align-items: center;
  gap: 20px;
}
.control.right {
  justify-content: flex-end;
}
.value {
  min-width: 40px;
  text-align: right;
}
</style>
