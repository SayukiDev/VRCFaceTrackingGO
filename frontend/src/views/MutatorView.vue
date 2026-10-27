<script setup lang="ts">
import { ref } from 'vue'
import { NCard, NCollapseTransition, NIcon, NSwitch, NText } from 'naive-ui'
import { ChevronDown20Regular } from '@vicons/fluent'
import { GetMutations, SetMutationActive } from '@go/app/app'
import type * as mixer from '@go/internal/mixer'
import MutationComponent from '../components/MutationComponent.vue'
import { useMutationText } from '../i18n'

const { mutationName, mutationDescription } = useMutationText()

const mutations = ref<mixer.StageState[]>([])
GetMutations().then((m) => (mutations.value = m))

const expanded = ref(new Set<string>())

function toggle(name: string) {
  if (!expanded.value.delete(name)) expanded.value.add(name)
}

function setActive(m: mixer.StageState, v: boolean) {
  m.active = v
  void SetMutationActive(m.name, v)
}
</script>

<template>
  <div class="cards">
    <n-card v-for="m in mutations" :key="m.name" size="small">
      <div class="row">
        <div class="summary" @click="toggle(m.name)">
          <n-icon class="chevron" :class="{ open: expanded.has(m.name) }" size="18"><chevron-down20-regular /></n-icon>
          <div class="info">
            <div class="name">{{ mutationName(m.name) }}</div>
            <n-text depth="3" class="desc">{{ mutationDescription(m.name, m.description) }}</n-text>
          </div>
        </div>
        <n-switch :value="m.active" @update:value="(v: boolean) => setActive(m, v)" />
      </div>
      <n-collapse-transition :show="expanded.has(m.name)">
        <div class="details">
          <mutation-component v-for="c in m.components" :key="c.name" :mutation="m.name" :component="c" />
        </div>
      </n-collapse-transition>
    </n-card>
  </div>
</template>

<style scoped>
.cards {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.row {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 4px 8px;
}
.summary {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 16px;
  cursor: pointer;
}
.chevron {
  flex-shrink: 0;
  transition: transform 0.2s;
  transform: rotate(-90deg);
}
.chevron.open {
  transform: rotate(0deg);
}
.info {
  flex: 1;
  min-width: 0;
}
.name {
  font-size: var(--md-sys-typescale-title-medium-size);
  line-height: var(--md-sys-typescale-title-medium-line-height);
  font-weight: var(--md-sys-typescale-title-medium-weight);
  letter-spacing: var(--md-sys-typescale-title-medium-tracking);
}
.desc {
  display: block;
  font-size: var(--md-sys-typescale-body-medium-size);
  line-height: var(--md-sys-typescale-body-medium-line-height);
}
.details {
  padding: 8px 8px 0;
}
</style>
