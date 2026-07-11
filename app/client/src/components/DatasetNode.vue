<script setup lang="ts">
import { computed, inject, type Ref } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import Sparkline from './Sparkline.vue'
import type { DatasetEntry } from '../types'

const props = defineProps<{ entry: DatasetEntry }>()

const selectedId = inject<Ref<string | null>>('selectedId')
const selected = computed(() => selectedId?.value === props.entry.id)

const slash = computed(() => props.entry.id.lastIndexOf('/'))
const group = computed(() => (slash.value >= 0 ? props.entry.id.slice(0, slash.value + 1) : ''))
const name = computed(() => (slash.value >= 0 ? props.entry.id.slice(slash.value + 1) : props.entry.id))

const meta = computed(() => {
  const parts: string[] = []
  if (typeof props.entry.row_count === 'number') {
    parts.push(`${props.entry.row_count.toLocaleString()} rows`)
  }
  if (props.entry.unit) parts.push(props.entry.unit)
  return parts.join(' · ')
})
</script>

<template>
  <div class="dataset-node" :class="{ selected }">
    <Handle type="target" :position="Position.Left" />
    <div class="head">
      <span class="group" :title="entry.id">{{ group || '·' }}</span>
      <span v-if="entry.origin" class="origin">{{ entry.origin }}</span>
    </div>
    <div class="name" :title="entry.id">{{ name }}</div>
    <div class="meta">{{ meta || 'no metadata' }}</div>
    <Sparkline :id="entry.id" class="spark" />
    <Handle type="source" :position="Position.Right" />
  </div>
</template>

<style scoped>
.dataset-node {
  width: 224px;
  height: 96px;
  display: flex;
  flex-direction: column;
  padding: 10px 12px 8px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 10px;
  box-shadow: var(--shadow);
  transition:
    box-shadow 120ms ease,
    border-color 120ms ease,
    transform 120ms ease;
}

.dataset-node:hover {
  transform: translateY(-1px);
  box-shadow: var(--shadow-raised);
}

.dataset-node.selected {
  border-color: var(--series);
  box-shadow:
    0 0 0 1px var(--series),
    var(--shadow-raised);
}

.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 15px;
}

.group {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--ink-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.origin {
  font-family: var(--font-mono);
  font-size: 9px;
  letter-spacing: 0.04em;
  color: var(--ink-secondary);
  background: var(--page);
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 1px 5px;
  white-space: nowrap;
}

.name {
  font-family: var(--font-mono);
  font-size: 14px;
  font-weight: 500;
  color: var(--ink);
  margin-top: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.meta {
  font-size: 11px;
  color: var(--ink-secondary);
  margin-top: 1px;
}

.spark {
  margin-top: auto;
}

:deep(.vue-flow__handle) {
  width: 6px;
  height: 6px;
  background: var(--baseline);
  border: none;
  min-width: 0;
  min-height: 0;
}
</style>
