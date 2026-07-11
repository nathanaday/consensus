import type { GraphResponse, SeriesResponse } from './types'

async function getJson<T>(url: string): Promise<T> {
  const res = await fetch(url)
  if (!res.ok) {
    let message = `${res.status} ${res.statusText}`
    try {
      const body = await res.json()
      if (body?.error) message = body.error
    } catch {
      /* keep the status line */
    }
    throw new Error(message)
  }
  return res.json()
}

export function fetchGraph(): Promise<GraphResponse> {
  return getJson('/api/graph')
}

export function fetchSeries(id: string): Promise<SeriesResponse> {
  return getJson(`/api/dataset/data?id=${encodeURIComponent(id)}`)
}

const sparkCache = new Map<string, Promise<number[]>>()

export function fetchSparkline(id: string): Promise<number[]> {
  let p = sparkCache.get(id)
  if (!p) {
    p = getJson<{ points: number[] }>(`/api/dataset/spark?id=${encodeURIComponent(id)}`).then(
      (r) => r.points,
    )
    p.catch(() => sparkCache.delete(id))
    sparkCache.set(id, p)
  }
  return p
}

export function invalidateSparklines(): void {
  sparkCache.clear()
}

// subscribeStoreEvents opens the SSE channel and reports connection state.
// EventSource retries transient drops itself, but gives up for good when the
// endpoint answers with an error (e.g. the dev proxy while the API restarts),
// so a closed source is rebuilt on a timer. Every open triggers onChange to
// resync whatever was missed while disconnected.
export function subscribeStoreEvents(
  onChange: () => void,
  onStateChange: (connected: boolean) => void,
): () => void {
  let source: EventSource | null = null
  let retry: number | null = null
  let stopped = false

  const connect = () => {
    source = new EventSource('/api/events')
    source.onopen = () => {
      onStateChange(true)
      onChange()
    }
    source.onerror = () => {
      onStateChange(false)
      if (source?.readyState === EventSource.CLOSED && !stopped && retry === null) {
        retry = window.setTimeout(() => {
          retry = null
          connect()
        }, 2500)
      }
    }
    source.addEventListener('store-changed', onChange)
  }
  connect()

  return () => {
    stopped = true
    if (retry !== null) window.clearTimeout(retry)
    source?.close()
  }
}
