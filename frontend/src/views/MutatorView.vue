<script setup lang="ts">
import { ref } from 'vue'
import { NCollapse, NCollapseItem, NSwitch, NText } from 'naive-ui'
import { GetMutations, SetMutationActive } from '../../wailsjs/go/app/App'
import type { mixer } from '../../wailsjs/go/models'
import MutationComponent from '../components/MutationComponent.vue'
import { useMutationText } from '../i18n'

const { mutationName, mutationDescription } = useMutationText()

const mutations = ref<mixer.StageState[]>([])
GetMutations().then((m) => (mutations.value = m))

function setActive(m: mixer.StageState, v: boolean) {
  m.active = v
  void SetMutationActive(m.name, v)
}
</script>

<template>
  <n-collapse display-directive="show" arrow-placement="right">
    <n-collapse-item v-for="m in mutations" :key="m.name" :name="m.name">
      <template #header>
        <div class="header" @click.stop>
          <n-switch :value="m.active" @update:value="(v: boolean) => setActive(m, v)" />
          <div class="text">
            <div>{{ mutationName(m.name) }}</div>
            <n-text depth="3" class="desc">{{ mutationDescription(m.name, m.description) }}</n-text>
          </div>
        </div>
      </template>
      <mutation-component v-for="c in m.components" :key="c.name" :mutation="m.name" :component="c" />
    </n-collapse-item>
  </n-collapse>
</template>

<style scoped>
.header {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 12px 10px;
}
.desc {
  font-size: 13px;
}
</style>
