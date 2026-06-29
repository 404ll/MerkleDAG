package graph

import (
	"fmt"
	"html"
	"sort"
	"strconv"
	"strings"

	"merkledag/object"
	"merkledag/store"
)

type Node struct {
	CID  string
	Type object.ObjectType
}

type Edge struct {
	From  string
	To    string
	Label string
}

type dag struct {
	nodes map[string]Node
	edges []Edge
}

type point struct {
	x int
	y int
}

const (
	canvasMargin = 40
	nodeWidth    = 170
	nodeHeight   = 46
	columnGap    = 90
	rowGap       = 32
)

func RenderDOT(rootCID string, st store.Store) (string, error) {
	g, err := build(rootCID, st)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("digraph MerkleDAG {\n")
	b.WriteString("  rankdir=LR;\n")

	nodes := sortedNodes(g.nodes)
	for _, node := range nodes {
		fmt.Fprintf(&b, "  %s [label=%s, shape=%s];\n",
			dotID(node.CID),
			dotString(fmt.Sprintf("%s\\n%s", node.Type, shortCID(node.CID))),
			dotShape(node.Type),
		)
	}

	for _, edge := range g.edges {
		fmt.Fprintf(&b, "  %s -> %s [label=%s];\n",
			dotID(edge.From),
			dotID(edge.To),
			dotString(edge.Label),
		)
	}

	b.WriteString("}\n")
	return b.String(), nil
}

func RenderHTML(rootCID string, st store.Store) (string, error) {
	g, err := build(rootCID, st)
	if err != nil {
		return "", err
	}

	positions := layout(rootCID, g)
	width, height := canvasSize(positions)

	var b strings.Builder
	b.WriteString("<!doctype html>\n")
	b.WriteString("<html lang=\"en\">\n")
	b.WriteString("<head>\n")
	b.WriteString("  <meta charset=\"utf-8\">\n")
	b.WriteString("  <title>Merkle DAG</title>\n")
	b.WriteString("  <style>\n")
	b.WriteString("    body { font-family: -apple-system, BlinkMacSystemFont, \"Segoe UI\", sans-serif; margin: 32px; color: #202124; background: #f7f7f4; }\n")
	b.WriteString("    h1 { font-size: 28px; margin: 0 0 8px; }\n")
	b.WriteString("    .root { margin: 0 0 24px; color: #5f6368; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }\n")
	b.WriteString("    .canvas { overflow: auto; border: 1px solid #d0d0ca; background: #fff; }\n")
	b.WriteString("    svg { display: block; }\n")
	b.WriteString("    line { stroke: #8d9189; stroke-width: 1.5; }\n")
	b.WriteString("    rect { stroke: #5f6368; stroke-width: 1.2; rx: 6; }\n")
	b.WriteString("    .tree { fill: #dff1ff; }\n")
	b.WriteString("    .list { fill: #fff1cc; }\n")
	b.WriteString("    .blob { fill: #e7f4df; }\n")
	b.WriteString("    text { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; fill: #202124; }\n")
	b.WriteString("    .type { font-size: 12px; font-weight: 700; text-transform: uppercase; }\n")
	b.WriteString("    .cid { font-size: 12px; }\n")
	b.WriteString("    .edge-label { font-size: 11px; fill: #8a4b08; }\n")
	b.WriteString("  </style>\n")
	b.WriteString("</head>\n")
	b.WriteString("<body>\n")
	b.WriteString("  <h1>Merkle DAG</h1>\n")
	fmt.Fprintf(&b, "  <p class=\"root\">Root CID: %s</p>\n", html.EscapeString(rootCID))
	fmt.Fprintf(&b, "  <div class=\"canvas\"><svg width=\"%d\" height=\"%d\" viewBox=\"0 0 %d %d\" role=\"img\" aria-label=\"Merkle DAG graph\">\n", width, height, width, height)
	for _, edge := range g.edges {
		from := positions[edge.From]
		to := positions[edge.To]
		x1 := from.x + nodeWidth
		y1 := from.y + nodeHeight/2
		x2 := to.x
		y2 := to.y + nodeHeight/2
		fmt.Fprintf(&b, "    <line x1=\"%d\" y1=\"%d\" x2=\"%d\" y2=\"%d\"></line>\n", x1, y1, x2, y2)
		fmt.Fprintf(&b, "    <text class=\"edge-label\" x=\"%d\" y=\"%d\">%s</text>\n", x2-78, y2-6, html.EscapeString(edge.Label))
	}
	for _, node := range sortedNodes(g.nodes) {
		pos := positions[node.CID]
		fmt.Fprintf(&b, "    <g title=\"%s\">\n", html.EscapeString(node.CID))
		fmt.Fprintf(&b, "      <rect class=\"%s\" x=\"%d\" y=\"%d\" width=\"%d\" height=\"%d\"></rect>\n", html.EscapeString(string(node.Type)), pos.x, pos.y, nodeWidth, nodeHeight)
		fmt.Fprintf(&b, "      <text class=\"type\" x=\"%d\" y=\"%d\">%s</text>\n", pos.x+10, pos.y+18, html.EscapeString(string(node.Type)))
		fmt.Fprintf(&b, "      <text class=\"cid\" x=\"%d\" y=\"%d\">%s</text>\n", pos.x+10, pos.y+36, html.EscapeString(shortCID(node.CID)))
		b.WriteString("    </g>\n")
	}
	b.WriteString("  </svg></div>\n")
	b.WriteString("</body>\n")
	b.WriteString("</html>\n")
	return b.String(), nil
}

func build(rootCID string, st store.Store) (dag, error) {
	g := dag{
		nodes: make(map[string]Node),
	}
	if err := walk(rootCID, st, &g, map[string]bool{}); err != nil {
		return dag{}, err
	}
	return g, nil
}

func walk(cid string, st store.Store, g *dag, visited map[string]bool) error {
	obj, err := st.GetObject(cid)
	if err != nil {
		return err
	}

	g.nodes[cid] = Node{CID: cid, Type: obj.Type}
	if visited[cid] {
		return nil
	}
	visited[cid] = true

	switch obj.Type {
	case object.TreeType:
		for _, link := range obj.Links {
			g.edges = append(g.edges, Edge{From: cid, To: link.CID, Label: link.Name})
			if err := walk(link.CID, st, g, visited); err != nil {
				return err
			}
		}
	case object.ListType:
		for i, link := range obj.Links {
			label := link.Name
			if label == "" {
				label = fmt.Sprintf("chunk-%d", i)
			}
			g.edges = append(g.edges, Edge{From: cid, To: link.CID, Label: label})
			if err := walk(link.CID, st, g, visited); err != nil {
				return err
			}
		}
	case object.BlobType:
		return nil
	default:
		return fmt.Errorf("unknown object type %q: %s", obj.Type, cid)
	}
	return nil
}

func sortedNodes(nodes map[string]Node) []Node {
	out := make([]Node, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, node)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CID < out[j].CID
	})
	return out
}

func layout(rootCID string, g dag) map[string]point {
	children := make(map[string][]Edge)
	for _, edge := range g.edges {
		children[edge.From] = append(children[edge.From], edge)
	}
	for from := range children {
		sort.Slice(children[from], func(i, j int) bool {
			if children[from][i].Label == children[from][j].Label {
				return children[from][i].To < children[from][j].To
			}
			return children[from][i].Label < children[from][j].Label
		})
	}

	positions := make(map[string]point, len(g.nodes))
	nextY := canvasMargin
	placeNode(rootCID, 0, children, positions, map[string]bool{}, &nextY)

	orphanCIDs := make([]string, 0)
	for cid := range g.nodes {
		if _, ok := positions[cid]; !ok {
			orphanCIDs = append(orphanCIDs, cid)
		}
	}
	sort.Strings(orphanCIDs)
	for _, cid := range orphanCIDs {
		positions[cid] = point{x: canvasMargin, y: nextY}
		nextY += nodeHeight + rowGap
	}
	return positions
}

func placeNode(cid string, depth int, children map[string][]Edge, positions map[string]point, visiting map[string]bool, nextY *int) int {
	if pos, ok := positions[cid]; ok {
		return pos.y
	}
	if visiting[cid] {
		y := *nextY
		*nextY += nodeHeight + rowGap
		positions[cid] = point{x: canvasMargin + depth*(nodeWidth+columnGap), y: y}
		return y
	}
	visiting[cid] = true
	defer delete(visiting, cid)

	edges := children[cid]
	if len(edges) == 0 {
		y := *nextY
		*nextY += nodeHeight + rowGap
		positions[cid] = point{x: canvasMargin + depth*(nodeWidth+columnGap), y: y}
		return y
	}

	firstY := placeNode(edges[0].To, depth+1, children, positions, visiting, nextY)
	lastY := firstY
	for _, edge := range edges[1:] {
		lastY = placeNode(edge.To, depth+1, children, positions, visiting, nextY)
	}
	y := (firstY + lastY) / 2
	positions[cid] = point{x: canvasMargin + depth*(nodeWidth+columnGap), y: y}
	return y
}

func canvasSize(positions map[string]point) (int, int) {
	width := canvasMargin*2 + nodeWidth
	height := canvasMargin*2 + nodeHeight
	for _, pos := range positions {
		if pos.x+nodeWidth+canvasMargin > width {
			width = pos.x + nodeWidth + canvasMargin
		}
		if pos.y+nodeHeight+canvasMargin > height {
			height = pos.y + nodeHeight + canvasMargin
		}
	}
	return width, height
}

func dotID(value string) string {
	return dotString(value)
}

func dotString(value string) string {
	return strconv.Quote(value)
}

func dotShape(t object.ObjectType) string {
	switch t {
	case object.TreeType:
		return "folder"
	case object.ListType:
		return "component"
	case object.BlobType:
		return "box"
	default:
		return "ellipse"
	}
}

func shortCID(cid string) string {
	if len(cid) <= 12 {
		return cid
	}
	return cid[:12]
}
