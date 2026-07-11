// Read-only view over the consensus dataset store: the catalog.json written by
// the MCP server plus one Parquet file per dataset. This app never writes to
// the store; the MCP server owns all mutations.
import { readFile, stat } from 'node:fs/promises'
import { homedir } from 'node:os'
import path from 'node:path'
import { asyncBufferFromFile, parquetReadObjects } from 'hyparquet'

export interface TimeRange {
  start: string
  end: string
}

// Catalog entries are produced by the Go MCP server. Older store formats may
// lack some fields, so everything beyond `id` is treated as optional.
export interface CatalogEntry {
  id: string
  kind?: string
  source_path?: string
  source_column?: string
  unit?: string
  created_at?: string
  timestamp_column?: string
  row_count?: number
  time_range?: TimeRange
  parent_id?: string
  origin?: string
  [key: string]: unknown
}

export interface Series {
  timestamps: number[]
  values: number[]
}

export function storeDir(): string {
  return process.env.CONSENSUS_STORE_DIR || path.join(homedir(), '.consensus', 'store')
}

export function catalogPath(): string {
  return path.join(storeDir(), 'catalog.json')
}

export async function readCatalog(): Promise<Record<string, CatalogEntry>> {
  let raw: string
  try {
    raw = await readFile(catalogPath(), 'utf8')
  } catch (err) {
    if ((err as NodeJS.ErrnoException).code === 'ENOENT') return {}
    throw err
  }
  return JSON.parse(raw)
}

// parquetPath maps a dataset id to its file. Ids come from the catalog, never
// straight from the request path, so a lookup miss is the only guard needed.
function parquetPath(id: string): string {
  return path.join(storeDir(), id + '.parquet')
}

export async function datasetFileSize(id: string): Promise<number | null> {
  try {
    return (await stat(parquetPath(id))).size
  } catch {
    return null
  }
}

function toMillis(v: unknown): number {
  if (v instanceof Date) return v.getTime()
  return Number(v)
}

// readSeries loads one dataset's rows as parallel timestamp/value arrays,
// sorted by time. The store schema is a single channel: timestamp + value.
export async function readSeries(id: string): Promise<Series> {
  const file = await asyncBufferFromFile(parquetPath(id))
  const rows = (await parquetReadObjects({ file })) as Array<{ timestamp: unknown; value: unknown }>
  const pairs = rows
    .map((r) => [toMillis(r.timestamp), Number(r.value)] as const)
    .filter(([t, v]) => Number.isFinite(t) && Number.isFinite(v))
    .sort((a, b) => a[0] - b[0])
  return {
    timestamps: pairs.map((p) => p[0]),
    values: pairs.map((p) => p[1]),
  }
}

// sparkline reduces a series to at most `buckets` mean values — just enough
// shape for the small in-node preview.
export function sparkline(values: number[], buckets = 60): number[] {
  if (values.length <= buckets) return values
  const out: number[] = []
  for (let i = 0; i < buckets; i++) {
    const start = Math.floor((i * values.length) / buckets)
    const end = Math.max(start + 1, Math.floor(((i + 1) * values.length) / buckets))
    let sum = 0
    for (let j = start; j < end; j++) sum += values[j]!
    out.push(sum / (end - start))
  }
  return out
}
