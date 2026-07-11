<script setup lang="ts">
import { nextTick, provide, ref, toRef, watch } from 'vue'
import { VueFlow, useVueFlow, type Node, type Edge } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import DatasetNode from './DatasetNode.vue'
import { layoutGraph } from '../layout'
import type { DatasetEntry, GraphResponse } from '../types'

const props = defineProps<{
  graph: GraphResponse
  selectedId: string | null
  focusRequest: { id: string; token: number } | null
}>()

const emit = defineEmits<{
  select: [id: string | null]
}>()

provide('selectedId', toRef(props, 'selectedId'))

const nodes = ref<Node<DatasetEntry>[]>([])
const edges = ref<Edge[]>([])

const { fitView } = useVueFlow()

// topologyKey changes only when the set of datasets or edges changes — data
// refreshes that alter nothing structural must not move the viewport.
function topologyKey(g: GraphResponse): string {
  return g.nodes.map((n) => n.id).join('|') + '||' + g.edges.map((e) => e.id).join('|')
}

let lastTopology = ''
watch(
  () => props.graph,
  async (g) => {
    const layout = layoutGraph(g.nodes, g.edges)
    nodes.value = layout.nodes
    edges.value = layout.edges
    const key = topologyKey(g)
    if (lastTopology && key !== lastTopology) {
      await nextTick()
      fitView({ duration: 400, padding: 0.15 })
    }
    lastTopology = key
  },
  { immediate: true },
)

watch(
  () => props.focusRequest,
  async (req) => {
    if (!req) return
    await nextTick()
    fitView({ nodes: [req.id], duration: 400, maxZoom: 1.1 })
  },
)
</script>

<template>
  <VueFlow
    :nodes="nodes"
    :edges="edges"
    :fit-view-on-init="true"
    :max-zoom="1.75"
    :min-zoom="0.2"
    :nodes-connectable="false"
    :edges-updatable="false"
    :delete-key-code="null"
    @node-click="emit('select', $event.node.id)"
    @pane-click="emit('select', null)"
  >
    <Background pattern-color="" :gap="24" :size="1.25" />
    <Controls :show-interactive="false" position="bottom-left" />
    <template #node-dataset="nodeProps">
      <DatasetNode :entry="nodeProps.data" />
    </template>
    <div v-if="graph.nodes.length === 0" class="empty">
      <p class="empty-title">The store is empty</p>
      <p class="empty-hint">
        Ingest a CSV from an MCP session and it appears here as it lands:
      </p>
      <code>ingest_csv path=/path/to/data.csv</code>
    </div>
  </VueFlow>
</template>

<style>
.vue-flow {
  background: var(--page);
}

.vue-flow__background circle {
  fill: var(--grid);
}

.vue-flow__edge-path {
  stroke: var(--baseline);
  stroke-width: 1.5;
}

.vue-flow__edge.selected .vue-flow__edge-path {
  stroke: var(--series);
}

.vue-flow__edge-text {
  fill: var(--ink-muted);
  font-family: var(--font-mono);
  font-size: 10px;
}

.vue-flow__edge-textbg {
  fill: var(--page);
}

.vue-flow__controls {
  box-shadow: var(--shadow);
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
}

.vue-flow__controls-button {
  background: var(--surface);
  border-bottom: 1px solid var(--border);
  width: 28px;
  height: 28px;
}

.vue-flow__controls-button:last-child {
  border-bottom: none;
}

.vue-flow__controls-button:hover {
  background: var(--page);
}

.vue-flow__controls-button svg {
  fill: var(--ink-secondary);
}

.empty {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  pointer-events: none;
  z-index: 4;
}

.empty-title {
  font-size: 16px;
  font-weight: 600;
  margin: 0;
}

.empty-hint {
  margin: 0;
  color: var(--ink-secondary);
  font-size: 13px;
}

.empty code {
  margin-top: 8px;
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--ink-secondary);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 6px 10px;
}
</style>
