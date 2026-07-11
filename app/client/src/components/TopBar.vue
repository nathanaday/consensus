<script setup lang="ts">
import { useTheme } from '../useTheme'

defineProps<{
  storeDir: string
  datasetCount: number
  live: boolean
}>()

const { theme, toggleTheme } = useTheme()
</script>

<template>
  <header class="topbar">
    <div class="brand">
      <span class="name">Consensus</span>
      <span class="divider" aria-hidden="true"></span>
      <span class="role">Store Explorer</span>
    </div>
    <div class="status">
      <span v-if="storeDir" class="store-path" :title="storeDir">{{ storeDir }}</span>
      <span class="count">{{ datasetCount }} dataset{{ datasetCount === 1 ? '' : 's' }}</span>
      <span class="live" :class="{ on: live }">
        <span class="dot" aria-hidden="true"></span>
        {{ live ? 'Live' : 'Connecting' }}
      </span>
      <button
        class="theme-toggle"
        :aria-label="theme === 'light' ? 'Switch to dark theme' : 'Switch to light theme'"
        @click="toggleTheme"
      >
        {{ theme === 'light' ? 'Dark' : 'Light' }}
      </button>
    </div>
  </header>
</template>

<style scoped>
.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 0 20px;
  height: 52px;
  background: var(--surface);
  border-bottom: 1px solid var(--border);
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  white-space: nowrap;
}

.name {
  font-weight: 600;
  font-size: 15px;
  letter-spacing: 0.01em;
}

.divider {
  width: 1px;
  height: 18px;
  background: var(--baseline);
}

.role {
  color: var(--ink-secondary);
  font-size: 13px;
}

.status {
  display: flex;
  align-items: center;
  gap: 16px;
  min-width: 0;
}

.store-path {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--ink-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  direction: rtl;
  max-width: 340px;
}

.count {
  font-size: 13px;
  color: var(--ink-secondary);
  white-space: nowrap;
}

.live {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--ink-muted);
  white-space: nowrap;
}

.live .dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--baseline);
}

.live.on {
  color: var(--ink-secondary);
}

.live.on .dot {
  background: var(--good);
}

.theme-toggle {
  font-size: 13px;
  color: var(--ink-secondary);
  padding: 5px 10px;
  border: 1px solid var(--border);
  border-radius: 6px;
}

.theme-toggle:hover {
  background: var(--page);
}
</style>
