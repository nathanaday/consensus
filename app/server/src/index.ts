import { watch, existsSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import Fastify from 'fastify'
import fastifyStatic from '@fastify/static'
import {
  readCatalog,
  readSeries,
  sparkline,
  datasetFileSize,
  storeDir,
  type CatalogEntry,
} from './store.js'

const PORT = Number(process.env.PORT || 4400)

const app = Fastify({ logger: { level: 'warn' } })

interface GraphNode extends CatalogEntry {
  size_bytes: number | null
}

interface GraphEdge {
  id: string
  source: string
  target: string
  origin: string
}

app.get('/api/store', async () => {
  const catalog = await readCatalog()
  return { dir: storeDir(), dataset_count: Object.keys(catalog).length }
})

app.get('/api/graph', async () => {
  const catalog = await readCatalog()
  const entries = Object.values(catalog).sort((a, b) => a.id.localeCompare(b.id))
  const nodes: GraphNode[] = await Promise.all(
    entries.map(async (e) => ({ ...e, size_bytes: await datasetFileSize(e.id) })),
  )
  const edges: GraphEdge[] = entries
    .filter((e) => e.parent_id && catalog[e.parent_id])
    .map((e) => ({
      id: `${e.parent_id}->${e.id}`,
      source: e.parent_id!,
      target: e.id,
      origin: e.origin || 'copy',
    }))
  return { dir: storeDir(), nodes, edges }
})

async function requireDataset(id: unknown): Promise<CatalogEntry | null> {
  if (typeof id !== 'string' || id === '') return null
  const catalog = await readCatalog()
  return catalog[id] ?? null
}

app.get<{ Querystring: { id?: string } }>('/api/dataset/data', async (req, reply) => {
  const entry = await requireDataset(req.query.id)
  if (!entry) {
    return reply.code(404).send({ error: `unknown dataset ${JSON.stringify(req.query.id ?? '')}` })
  }
  try {
    const series = await readSeries(entry.id)
    return {
      id: entry.id,
      unit: entry.unit ?? null,
      count: series.values.length,
      timestamps: series.timestamps,
      values: series.values,
    }
  } catch (err) {
    return reply.code(500).send({ error: `read parquet: ${(err as Error).message}` })
  }
})

app.get<{ Querystring: { id?: string } }>('/api/dataset/spark', async (req, reply) => {
  const entry = await requireDataset(req.query.id)
  if (!entry) {
    return reply.code(404).send({ error: `unknown dataset ${JSON.stringify(req.query.id ?? '')}` })
  }
  try {
    const series = await readSeries(entry.id)
    return { id: entry.id, points: sparkline(series.values) }
  } catch (err) {
    return reply.code(500).send({ error: `read parquet: ${(err as Error).message}` })
  }
})

// --- live updates -----------------------------------------------------------
// The catalog write is the MCP store's commit point, so watching catalog.json
// is enough to know when the graph changed. Clients hold an SSE connection and
// refetch on every "store-changed" event.

const sseClients = new Set<NodeJS.WritableStream>()

app.get('/api/events', (req, reply) => {
  reply.raw.writeHead(200, {
    'content-type': 'text/event-stream',
    'cache-control': 'no-cache',
    connection: 'keep-alive',
  })
  reply.raw.write('retry: 2000\n\n')
  sseClients.add(reply.raw)
  const keepalive = setInterval(() => reply.raw.write(': keepalive\n\n'), 25_000)
  req.raw.on('close', () => {
    clearInterval(keepalive)
    sseClients.delete(reply.raw)
  })
})

function broadcastStoreChanged() {
  for (const client of sseClients) {
    client.write('event: store-changed\ndata: {}\n\n')
  }
}

function watchStore() {
  let timer: NodeJS.Timeout | null = null
  try {
    watch(storeDir(), (_event, filename) => {
      if (filename && filename !== 'catalog.json') return
      if (timer) clearTimeout(timer)
      timer = setTimeout(broadcastStoreChanged, 250)
    })
  } catch (err) {
    app.log.warn(`store watch unavailable: ${(err as Error).message}`)
  }
}

// --- static client ----------------------------------------------------------

const clientDist = path.join(path.dirname(fileURLToPath(import.meta.url)), '../../client/dist')
if (existsSync(clientDist)) {
  app.register(fastifyStatic, { root: clientDist })
}

app.listen({ port: PORT, host: '127.0.0.1' }).then(() => {
  watchStore()
  console.log(`consensus app server on http://localhost:${PORT} (store: ${storeDir()})`)
})
