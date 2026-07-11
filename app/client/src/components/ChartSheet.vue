<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import * as echarts from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, DataZoomComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { fetchSeries } from '../api'
import { useTheme } from '../useTheme'
import type { DatasetEntry, SeriesResponse } from '../types'

echarts.use([LineChart, GridComponent, TooltipComponent, DataZoomComponent, CanvasRenderer])

const props = defineProps<{ entry: DatasetEntry }>()
const emit = defineEmits<{ close: [] }>()

const { theme } = useTheme()

const container = ref<HTMLDivElement | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const series = ref<SeriesResponse | null>(null)

let chart: echarts.ECharts | null = null
let resizeObserver: ResizeObserver | null = null

function cssVar(name: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

const numberFormat = new Intl.NumberFormat(undefined, { maximumSignificantDigits: 5 })

const timeFormat = new Intl.DateTimeFormat('en-US', {
  year: 'numeric',
  month: 'short',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
  timeZone: 'UTC',
})

const stats = computed(() => {
  const s = series.value
  if (!s || s.values.length === 0) return null
  let min = Infinity
  let max = -Infinity
  let sum = 0
  for (const v of s.values) {
    if (v < min) min = v
    if (v > max) max = v
    sum += v
  }
  return {
    count: s.values.length.toLocaleString(),
    min: numberFormat.format(min),
    max: numberFormat.format(max),
    mean: numberFormat.format(sum / s.values.length),
  }
})

// Style-only option fragment, re-applied (merged) on theme change so the
// data zoom window survives a theme switch.
function styleOption(): echarts.EChartsCoreOption {
  const ink = cssVar('--ink')
  const muted = cssVar('--ink-muted')
  const grid = cssVar('--grid')
  const baseline = cssVar('--baseline')
  const surface = cssVar('--surface')
  const seriesColor = cssVar('--series')
  const border = cssVar('--border')
  const fontSans = cssVar('--font-sans')
  const fontMono = cssVar('--font-mono')
  const unit = props.entry.unit

  return {
    xAxis: {
      axisLine: { lineStyle: { color: baseline } },
      axisLabel: { color: muted, fontFamily: fontSans, fontSize: 11 },
    },
    yAxis: {
      axisLabel: { color: muted, fontFamily: fontSans, fontSize: 11 },
      splitLine: { lineStyle: { color: grid, width: 1 } },
    },
    tooltip: {
      backgroundColor: surface,
      borderColor: border,
      borderWidth: 1,
      padding: [8, 12],
      textStyle: { color: ink, fontFamily: fontSans, fontSize: 12 },
      extraCssText: 'box-shadow: var(--shadow-raised); border-radius: 8px;',
      axisPointer: { lineStyle: { color: baseline, width: 1 } },
      formatter: (params: unknown) => {
        const p = (params as Array<{ value: [number, number] }>)[0]
        if (!p) return ''
        const [t, v] = p.value
        const value = numberFormat.format(v)
        const suffix = unit ? ` <span style="color:${muted}">${escapeHtml(unit)}</span>` : ''
        return (
          `<div style="font-weight:600;font-size:13px">${value}${suffix}</div>` +
          `<div style="color:${muted};font-size:11px;margin-top:2px">${timeFormat.format(t)} UTC</div>`
        )
      },
    },
    dataZoom: [
      {},
      {
        borderColor: border,
        fillerColor: 'rgba(137, 135, 129, 0.12)',
        handleStyle: { color: surface, borderColor: baseline },
        moveHandleStyle: { color: baseline },
        dataBackground: {
          lineStyle: { color: baseline, width: 1 },
          areaStyle: { color: grid, opacity: 0.4 },
        },
        selectedDataBackground: {
          lineStyle: { color: seriesColor, width: 1 },
          areaStyle: { color: seriesColor, opacity: 0.1 },
        },
        textStyle: { color: muted, fontFamily: fontMono, fontSize: 10 },
      },
    ],
    series: [
      {
        color: seriesColor,
        areaStyle: { color: seriesColor, opacity: 0.08 },
      },
    ],
  }
}

function baseOption(data: SeriesResponse): echarts.EChartsCoreOption {
  const points = data.timestamps.map((t, i) => [t, data.values[i]])
  return {
    animation: true,
    animationDuration: 300,
    animationDurationUpdate: 120,
    grid: { left: 64, right: 24, top: 20, bottom: 64 },
    xAxis: { type: 'time', axisTick: { show: false }, splitLine: { show: false } },
    yAxis: { type: 'value', scale: true, axisLine: { show: false }, axisTick: { show: false } },
    tooltip: { trigger: 'axis', axisPointer: { type: 'line', snap: true } },
    dataZoom: [
      {
        type: 'inside',
        zoomOnMouseWheel: true,
        moveOnMouseMove: true,
        moveOnMouseWheel: false,
        throttle: 40,
      },
      { type: 'slider', height: 20, bottom: 12 },
    ],
    series: [
      {
        type: 'line',
        name: data.id,
        data: points,
        showSymbol: false,
        sampling: 'lttb',
        lineStyle: { width: 2, cap: 'round', join: 'round' },
      },
    ],
  }
}

async function load() {
  loading.value = true
  error.value = null
  try {
    const data = await fetchSeries(props.entry.id)
    series.value = data
    if (chart) {
      chart.setOption(baseOption(data), { notMerge: true })
      chart.setOption(styleOption())
    }
  } catch (err) {
    error.value = (err as Error).message
    series.value = null
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  if (!container.value) return
  chart = echarts.init(container.value)
  resizeObserver = new ResizeObserver(() => chart?.resize())
  resizeObserver.observe(container.value)
  load()
})

onUnmounted(() => {
  resizeObserver?.disconnect()
  chart?.dispose()
  chart = null
})

watch(() => props.entry.id, load)
watch(theme, () => {
  // Wait a frame so the new CSS variables are in effect before reading them.
  requestAnimationFrame(() => chart?.setOption(styleOption()))
})
</script>

<template>
  <section class="chart-sheet" aria-label="Time series chart">
    <header class="head">
      <div class="title">
        <span class="name">{{ entry.id }}</span>
        <span v-if="entry.unit" class="unit">{{ entry.unit }}</span>
      </div>
      <dl v-if="stats" class="stats">
        <div class="stat"><dt>points</dt><dd>{{ stats.count }}</dd></div>
        <div class="stat"><dt>min</dt><dd>{{ stats.min }}</dd></div>
        <div class="stat"><dt>max</dt><dd>{{ stats.max }}</dd></div>
        <div class="stat"><dt>mean</dt><dd>{{ stats.mean }}</dd></div>
      </dl>
      <button class="close" aria-label="Close chart" @click="emit('close')">&#215;</button>
    </header>
    <div class="plot-wrap">
      <div ref="container" class="plot" :class="{ dimmed: loading }"></div>
      <div v-if="error" class="plot-error" role="alert">
        <strong>Can't load this dataset.</strong>
        <span>{{ error }}</span>
        <button class="retry" @click="load">Retry</button>
      </div>
    </div>
  </section>
</template>

<style scoped>
.chart-sheet {
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: var(--surface);
  border-top: 1px solid var(--border);
}

.head {
  display: flex;
  align-items: center;
  gap: 24px;
  padding: 10px 20px;
  border-bottom: 1px solid var(--border);
}

.title {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.name {
  font-family: var(--font-mono);
  font-size: 14px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.unit {
  font-size: 11px;
  color: var(--ink-secondary);
  background: var(--page);
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 1px 6px;
  white-space: nowrap;
}

.stats {
  display: flex;
  gap: 20px;
  margin: 0;
  margin-left: auto;
}

.stat {
  display: flex;
  align-items: baseline;
  gap: 6px;
}

.stat dt {
  font-size: 11px;
  color: var(--ink-muted);
}

.stat dd {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
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

.plot-wrap {
  position: relative;
  flex: 1;
  min-height: 0;
}

.plot {
  position: absolute;
  inset: 0;
  transition: opacity 150ms ease;
}

.plot.dimmed {
  opacity: 0.35;
}

.plot-error {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  font-size: 13px;
  color: var(--ink-secondary);
}

.plot-error strong {
  color: var(--ink);
  font-weight: 600;
}

.retry {
  color: var(--series);
  font-weight: 500;
}
</style>
