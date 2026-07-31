import type { Edge, Node } from "@xyflow/react";
import dagre from "dagre";

const NODE_WIDTH = 250;
const NODE_HEIGHT = 80;

export function getLayoutElements<T extends Record<string, unknown>>(
  nodes: Node<T>[],
  edges: Edge[]
): { nodes: Node<T>[]; edges: Edge[] } {
  const g = new dagre.graphlib.Graph().setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir: "TB", nodesep: 80, ranksep: 120 });

  nodes.forEach((n) =>
    g.setNode(n.id, { width: NODE_WIDTH, height: NODE_HEIGHT })
  );
  edges.forEach((e) => g.setEdge(e.source, e.target));

  dagre.layout(g);

  nodes.forEach((n) => {
    const nodeWithPos = g.node(n.id);
    n.position = {
      x: nodeWithPos.x - NODE_WIDTH / 2,
      y: nodeWithPos.y - NODE_HEIGHT / 2,
    };
  });

  return { nodes, edges };
}
