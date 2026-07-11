<script setup lang="ts">
import { computed } from 'vue'
import JsonView from './JsonView.vue'
import type { DatasetEntry, GraphResponse } from '../types'

const props = defineProps<{
  entry: DatasetEntry
  graph: GraphResponse
}>()

const emit = defineEmits<{
  close: []
  focusNode: [id: string]
  viewData: [id: string]
}>()

const slash = computed(() => props.entry.id.lastIndexOf('/'))
const group = computed(() => (slash.value >= 0 ? props.entry.id.slice(0, slash.value + 1) : ''))
const name = computed(() =>
  slash.value >= 0 ? props.entry.id.slice(slash.value + 1) : props.entry.id,
)

const parentId = computed(() => props.entry.parent_id || null)
const childIds = computed(() =>
  props.graph.edges.filter((e) => e.source === props.entry.id).map((e) => e.target),
)

// The JSON section shows the catalog entry exactly as stored, with the
// server-computed file size alongside.
const metadata = computed(() => {
  const { size_bytes, ...entry } = props.entry
  return { ...entry, size_bytes }
})
</script>

<template>
  <aside class="inspector" aria-label="Dataset inspector">
    <header class="head">
      <div class="title">
        <span v-if="group" class="group">{{ group }}</span>
        <h2 class="name">{{ name }}</h2>
      </div>
      <button class="close" aria-label="Close inspector" @click="emit('close')">&#215;</button>
    </header>

    <div class="body">
      <button class="view-data" @click="emit('viewData', entry.id)">View data</button>

      <section v-if="parentId || childIds.length" class="section">
        <h3 class="section-title">Lineage</h3>
        <div v-if="parentId" class="lineage-row">
          <span class="lineage-label">parent</span>
          <button class="chip" @click="emit('focusNode', parentId)">{{ parentId }}</button>
        </div>
        <div v-if="childIds.length" class="lineage-row">
          <span class="lineage-label">children</span>
          <div class="chips">
            <button v-for="id in childIds" :key="id" class="chip" @click="emit('focusNode', id)">
              {{ id }}
            </button>
          </div>
        </div>
      </section>
      <section v-else class="section">
        <h3 class="section-title">Lineage</h3>
        <p class="lineage-none">Root dataset with no derived copies yet.</p>
      </section>

      <section class="section">
        <h3 class="section-title">Metadata</h3>
        <JsonView :value="metadata" />
      </section>
    </div>
  </aside>
</template>

<style scoped>
.inspector {
  width: 360px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  background: var(--surface);
  border-left: 1px solid var(--border);
  min-height: 0;
}

.head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 20px 12px;
  border-bottom: 1px solid var(--border);
}

.title {
  min-width: 0;
}

.group {
  display: block;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--ink-muted);
}

.name {
  margin: 2px 0 0;
  font-family: var(--font-mono);
  font-size: 16px;
  font-weight: 600;
  word-break: break-all;
}

.close {
  font-size: 20px;
  line-height: 1;
  color: var(--ink-muted);
  padding: 2px 6px;
  border-radius: 6px;
}

.close:hover {
  color: var(--ink);
  background: var(--page);
}

.body {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px 24px;
}

.view-data {
  width: 100%;
  padding: 9px 0;
  background: var(--series);
  color: #ffffff;
  font-weight: 500;
  font-size: 13.5px;
  border-radius: 8px;
  transition: filter 120ms ease;
}

.view-data:hover {
  filter: brightness(1.08);
}

.section {
  margin-top: 22px;
}

.section-title {
  margin: 0 0 10px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--ink-muted);
}

.lineage-row {
  display: flex;
  gap: 10px;
  align-items: baseline;
  margin-bottom: 8px;
}

.lineage-label {
  flex-shrink: 0;
  width: 58px;
  font-size: 12px;
  color: var(--ink-muted);
}

.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  min-width: 0;
}

.chip {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--ink-secondary);
  background: var(--page);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 3px 8px;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chip:hover {
  color: var(--series);
  border-color: var(--series);
}

.lineage-none {
  margin: 0;
  font-size: 13px;
  color: var(--ink-secondary);
}
</style>
