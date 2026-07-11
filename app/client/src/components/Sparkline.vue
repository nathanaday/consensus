<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { fetchSparkline } from '../api'

const props = defineProps<{ id: string }>()

const W = 200
const H = 24
const PAD = 2

const points = ref<string | null>(null)
const failed = ref(false)

async function load() {
  points.value = null
  failed.value = false
  try {
    const values = await fetchSparkline(props.id)
    if (values.length < 2) {
      failed.value = true
      return
    }
    const min = Math.min(...values)
    const max = Math.max(...values)
    const span = max - min || 1
    points.value = values
      .map((v, i) => {
        const x = PAD + (i * (W - 2 * PAD)) / (values.length - 1)
        const y = H - PAD - ((v - min) * (H - 2 * PAD)) / span
        return `${x.toFixed(1)},${y.toFixed(1)}`
      })
      .join(' ')
  } catch {
    failed.value = true
  }
}

onMounted(load)
watch(() => props.id, load)
</script>

<template>
  <svg class="sparkline" :viewBox="`0 0 ${W} ${H}`" preserveAspectRatio="none" aria-hidden="true">
    <polyline
      v-if="points"
      :points="points"
      fill="none"
      stroke="var(--series)"
      stroke-width="1.5"
      stroke-linejoin="round"
      stroke-linecap="round"
      vector-effect="non-scaling-stroke"
    />
    <line
      v-else-if="failed"
      :x1="PAD"
      :y1="H / 2"
      :x2="W - PAD"
      :y2="H / 2"
      stroke="var(--grid)"
      stroke-width="1.5"
      stroke-dasharray="3 4"
    />
  </svg>
</template>

<style scoped>
.sparkline {
  display: block;
  width: 100%;
  height: 24px;
}
</style>
