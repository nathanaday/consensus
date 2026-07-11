export interface TimeRange {
  start: string
  end: string
}

// Mirrors the server's CatalogEntry; older store formats may omit fields.
export interface DatasetEntry {
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
  size_bytes: number | null
  [key: string]: unknown
}

export interface GraphEdgeData {
  id: string
  source: string
  target: string
  origin: string
}

export interface GraphResponse {
  dir: string
  nodes: DatasetEntry[]
  edges: GraphEdgeData[]
}

export interface SeriesResponse {
  id: string
  unit: string | null
  count: number
  timestamps: number[]
  values: number[]
}
