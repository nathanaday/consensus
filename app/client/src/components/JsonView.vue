<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ value: unknown }>()

interface Token {
  text: string
  cls: string
}

// Tokenize pretty-printed JSON into typed spans. Rendering tokens as text
// nodes (never v-html) keeps store-supplied strings inert.
const TOKEN_RE = /"(?:\\.|[^\\"])*"(?:\s*:)?|\btrue\b|\bfalse\b|\bnull\b|-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?/g

const tokens = computed<Token[]>(() => {
  const text = JSON.stringify(props.value, null, 2) ?? ''
  const out: Token[] = []
  let last = 0
  for (const match of text.matchAll(TOKEN_RE)) {
    if (match.index > last) out.push({ text: text.slice(last, match.index), cls: 'punct' })
    const t = match[0]
    let cls = 'number'
    if (t.startsWith('"')) cls = t.endsWith(':') ? 'key' : 'string'
    else if (t === 'true' || t === 'false' || t === 'null') cls = 'literal'
    out.push({ text: t, cls })
    last = match.index + t.length
  }
  if (last < text.length) out.push({ text: text.slice(last), cls: 'punct' })
  return out
})
</script>

<template>
  <pre class="json-view"><span v-for="(t, i) in tokens" :key="i" :class="t.cls">{{ t.text }}</span></pre>
</template>

<style scoped>
.json-view {
  margin: 0;
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.key {
  color: var(--ink);
  font-weight: 500;
}

.string {
  color: var(--ink-secondary);
}

.number,
.literal {
  color: var(--series);
}

.punct {
  color: var(--ink-muted);
}
</style>
