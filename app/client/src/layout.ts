import dagre from 'dagre'
import type { Node, Edge } from '@vue-flow/core'
import type { DatasetEntry, GraphEdgeData } from './types'

export const NODE_WIDTH = 224
export const NODE_HEIGHT = 96

// layoutGraph arranges the lineage forest left-to-right: roots in the first
// rank, each derived dataset one rank right of its parent.
export function layoutGraph(entries: DatasetEntry[], edgeData: GraphEdgeData[]): {
  nodes: Node<DatasetEntry>[]
  edges: Edge[]
} {
  const g = new dagre.graphlib.Graph()
  g.setGraph({ rankdir: 'LR', nodesep: 28, ranksep: 140, marginx: 40, marginy: 40 })
  g.setDefaultEdgeLabel(() => ({}))

  for (const entry of entries) {
    g.setNode(entry.id, { width: NODE_WIDTH, height: NODE_HEIGHT })
  }
  for (const edge of edgeData) {
    g.setEdge(edge.source, edge.target)
  }
  dagre.layout(g)

  const nodes: Node<DatasetEntry>[] = entries.map((entry) => {
    const pos = g.node(entry.id)
    return {
      id: entry.id,
      type: 'dataset',
      position: { x: pos.x - NODE_WIDTH / 2, y: pos.y - NODE_HEIGHT / 2 },
      data: entry,
    }
  })

  const edges: Edge[] = edgeData.map((e) => ({
    id: e.id,
    source: e.source,
    target: e.target,
    label: e.origin,
  }))

  return { nodes, edges }
}
