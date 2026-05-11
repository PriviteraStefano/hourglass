import type {Unit, UnitTreeNode} from '@/types/unit.ts'

export function flattenTree(nodes: UnitTreeNode[]): Unit[] {
  const result: Unit[] = []

  function walk(node: UnitTreeNode) {
    result.push(node.unit)
    node.children?.forEach(walk)
  }

  nodes.forEach(walk)
  return result
}

export function getDescendants(node: UnitTreeNode): string[] {
  const result: string[] = []

  function walk(n: UnitTreeNode) {
    result.push(n.unit.id)
    n.children?.forEach(walk)
  }

  walk(node)
  return result
}

export function getDescendantIds(unitId: string, nodes: UnitTreeNode[]): Set<string> {
  const ids = new Set<string>()
  const target = findNode(nodes, unitId)
  if (!target) return ids

  function walk(n: UnitTreeNode) {
    ids.add(n.unit.id)
    n.children?.forEach(walk)
  }

  walk(target)
  return ids
}

export function findNode(nodes: UnitTreeNode[], id: string): UnitTreeNode | undefined {
  for (const n of nodes) {
    if (n.unit.id === id) return n
    const found = n.children ? findNode(n.children, id) : undefined
    if (found) return found
  }
  return undefined
}
