<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import TopBar from './components/TopBar.vue'
import GraphCanvas from './components/GraphCanvas.vue'
import InspectorPanel from './components/InspectorPanel.vue'
import ChartSheet from './components/ChartSheet.vue'
import { fetchGraph, invalidateSparklines, subscribeStoreEvents } from './api'
import type { GraphResponse } from './types'

const graph = ref<GraphResponse | null>(null)
const loadError = ref<string | null>(null)
const live = ref(false)
const selectedId = ref<string | null>(null)
const chartId = ref<string | null>(null)
const focusRequest = ref<{ id: string; token: number } | null>(null)

const selectedEntry = computed(
  () => graph.value?.nodes.find((n) => n.id === selectedId.value) ?? null,
)
const chartEntry = computed(() => graph.value?.nodes.find((n) => n.id === chartId.value) ?? null)

async function refresh() {
  try {
    const next = await fetchGraph()
    graph.value = next
    loadError.value = null
    const ids = new Set(next.nodes.map((n) => n.id))
    if (selectedId.value && !ids.has(selectedId.value)) selectedId.value = null
    if (chartId.value && !ids.has(chartId.value)) chartId.value = null
  } catch (err) {
    loadError.value = (err as Error).message
  }
}

function onStoreChanged() {
  invalidateSparklines()
  refresh()
}

function onKeydown(e: KeyboardEvent) {
  if (e.key !== 'Escape') return
  if (chartId.value) chartId.value = null
  else if (selectedId.value) selectedId.value = null
}

let unsubscribe: (() => void) | null = null
onMounted(() => {
  refresh()
  unsubscribe = subscribeStoreEvents(onStoreChanged, (connected) => (live.value = connected))
  window.addEventListener('keydown', onKeydown)
})
onUnmounted(() => {
  unsubscribe?.()
  window.removeEventListener('keydown', onKeydown)
})

function selectNode(id: string | null) {
  selectedId.value = id
}

let focusToken = 0
function focusNode(id: string) {
  selectedId.value = id
  focusRequest.value = { id, token: ++focusToken }
}
</script>

<template>
  <div class="app" :class="{ 'chart-open': chartEntry }">
    <TopBar
      :store-dir="graph?.dir ?? ''"
      :dataset-count="graph?.nodes.length ?? 0"
      :live="live"
    />
    <main class="content">
      <div class="canvas-pane">
        <GraphCanvas
          v-if="graph"
          :graph="graph"
          :selected-id="selectedId"
          :focus-request="focusRequest"
          @select="selectNode"
        />
        <div v-if="loadError" class="load-error" role="alert">
          <strong>Can't reach the store.</strong>
          <span>{{ loadError }}</span>
          <button class="retry" @click="refresh">Retry</button>
        </div>
      </div>
      <InspectorPanel
        v-if="selectedEntry"
        :entry="selectedEntry"
        :graph="graph!"
        @close="selectNode(null)"
        @focus-node="focusNode"
        @view-data="chartId = $event"
      />
    </main>
    <ChartSheet v-if="chartEntry" :entry="chartEntry" @close="chartId = null" />
  </div>
</template>

<style scoped>
.app {
  height: 100%;
  display: grid;
  grid-template-rows: auto 1fr;
}

.app.chart-open {
  grid-template-rows: auto 1fr minmax(280px, 38vh);
}

.content {
  display: flex;
  min-height: 0;
}

.canvas-pane {
  position: relative;
  flex: 1;
  min-width: 0;
}

.load-error {
  position: absolute;
  top: 16px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  gap: 10px;
  align-items: baseline;
  padding: 10px 14px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 8px;
  box-shadow: var(--shadow);
  font-size: 13px;
  color: var(--ink-secondary);
}

.load-error strong {
  color: var(--ink);
  font-weight: 600;
}

.retry {
  color: var(--series);
  font-weight: 500;
}
</style>
