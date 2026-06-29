# DAG Visualization Design

## Goal

Add a simple DAG visualization feature for the local Merkle DAG store. The feature should help users inspect object relationships from a root CID during demos, debugging, and coursework review.

## Scope

Implement two output formats:

- DOT, for Graphviz rendering.
- Self-contained HTML, for opening directly in a browser.

JSON export is out of scope for this iteration.

## Command

Add a CLI command:

```bash
mdag graph <root-cid> <format>
```

Supported formats:

```bash
mdag graph <root-cid> dot
mdag graph <root-cid> html
```

The command writes generated content to stdout. Users can redirect output:

```bash
mdag graph <root-cid> dot > dag.dot
mdag graph <root-cid> html > dag.html
```

## Architecture

Add a new package:

```text
graph/graph.go
```

The package traverses stored objects from a root CID and builds an internal graph:

```go
type Node struct {
    CID  string
    Type object.ObjectType
}

type Edge struct {
    From  string
    To    string
    Label string
}
```

Public rendering functions:

```go
func RenderDOT(rootCID string, st store.Store) (string, error)
func RenderHTML(rootCID string, st store.Store) (string, error)
```

Traversal rules:

- Tree objects link to children using the original link name.
- List objects link to chunks using the link name when present, otherwise a generated chunk label.
- Blob objects are leaves.
- Already visited CIDs are not traversed again, preventing repeated work if a CID is shared.

## Output

DOT output starts with:

```dot
digraph MerkleDAG {
```

Nodes include object type and a shortened CID label. Edges include the link label.

HTML output is self-contained and uses no external JavaScript or CSS. It renders a readable relationship view with:

- Root CID.
- Node type.
- Short CID plus full CID in a title attribute.
- Parent-to-child links with labels.

The first HTML version may use a structured tree/list layout rather than a force-directed visual graph. The priority is clarity and zero dependencies.

## Error Handling

If a CID cannot be loaded, return the store error.

If the format is unsupported, the CLI returns a usage-style error.

## Tests

Add graph package tests that create a demo DAG using the existing importer and file store pattern. Cover:

- DOT output contains the graph header.
- DOT output contains at least one node and one labeled edge.
- HTML output contains a valid HTML skeleton.
- HTML output includes tree, list, and blob object types.
- Shared or repeated links do not cause infinite traversal.

