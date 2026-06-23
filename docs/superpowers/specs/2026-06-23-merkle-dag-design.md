# Merkle DAG Course Project Design

## Goal

Build a local, simplified Merkle DAG file storage and path resolution system in Go.

The project should be easy to compile, run, demonstrate, and explain in a course defense. It will implement the required Blob and Tree objects, plus a small List extension for chunked files. The system is not an IPFS clone; it focuses on content addressing, HashLink references, Tree path resolution, and file reading.

## Scope

Required features:

- Encode ordinary small files as Blob objects.
- Encode directories as Tree objects.
- Generate CID as `hex(SHA256(serializedObject))`.
- Persist objects under `./data/objects/<cid>.json`.
- Import files and directories recursively with `mdag add`.
- Resolve paths from a root CID with `mdag resolve`.
- Read file content with `mdag cat`.
- Report clear errors for missing CID, missing path component, non-Tree path traversal, non-file cat target, and decode failures.

Extension features:

- Encode files larger than 1024 bytes as List objects containing ordered Blob chunks.
- Read List objects by recursively reading and concatenating linked chunks.
- Add `mdag ls` for listing direct children of a Tree.
- Recompute CID after loading an object to detect object file tampering.

Out of scope:

- Real IPFS networking, DHT, Bitswap, provider discovery, IPNS, or DNSLink.
- Full CIDv0/CIDv1, multibase, multicodec, or multihash standards.
- UnixFS protobuf, HAMT directories, concurrent download, cache layers, GUI, or HTTP gateway.

## Project Structure

```text
cmd/mdag/main.go
object/object.go
object/codec.go
store/store.go
store/file.go
importer/importer.go
resolver/resolver.go
testdata/demo/
README.md
```

Responsibilities:

- `object`: defines object types and deterministic JSON encoding/CID generation.
- `store`: owns object persistence and integrity verification.
- `importer`: converts local files/directories into Merkle DAG objects.
- `resolver`: resolves Tree paths, reads Blob/List content, and lists Tree entries.
- `cmd/mdag`: parses CLI commands and prints user-facing output.

## Object Model

The project uses one unified object shape:

```go
type ObjectType string

const (
    BlobType ObjectType = "blob"
    TreeType ObjectType = "tree"
    ListType ObjectType = "list"
)

type Link struct {
    Name string `json:"name,omitempty"`
    CID  string `json:"cid"`
    Size int64  `json:"size,omitempty"`
}

type Object struct {
    Type  ObjectType `json:"type"`
    Data  []byte     `json:"data,omitempty"`
    Links []Link     `json:"links,omitempty"`
}
```

Meaning:

- Blob stores file bytes in `Data`.
- Tree stores direct child entries in `Links`; each Link has a child name and CID.
- List stores ordered chunk links in `Links`; Link names are optional because order matters more than names.

Tree links represent HashLinks from parent directories to child objects. List links represent HashLinks from a large file object to its chunk Blob objects.

## CID And Encoding

CID generation:

1. Serialize the object to deterministic JSON.
2. Compute SHA-256 over the serialized bytes.
3. Encode the digest as a lowercase hexadecimal string.

Important rule: the same logical object must produce the same serialized bytes and the same CID. Tree links will be sorted by name before storing so that importing the same directory twice produces the same root CID.

## Storage

The `Store` interface:

```go
type Store interface {
    PutObject(obj object.Object) (string, error)
    GetObject(cid string) (object.Object, error)
}
```

The file store writes each object as:

```text
./data/objects/<cid>.json
```

`PutObject` computes the CID, creates the object directory if needed, and writes the serialized object. Re-saving the same object is allowed and should produce the same CID.

`GetObject` reads the JSON file, decodes the object, recomputes its CID, and checks that the recomputed CID matches the requested CID. If not, it returns an integrity error.

## Import Flow

`AddPath(localPath, store)` returns the root CID for a file or directory.

For a directory:

1. Read direct directory entries.
2. Recursively call `AddPath` for each child.
3. Create one Tree Link per child with child name, CID, and size when useful.
4. Sort links by name.
5. Save the Tree and return its CID.

For a file:

1. If file size is less than or equal to 1024 bytes, read all bytes and save one Blob.
2. If file size is greater than 1024 bytes, split it into 1024-byte chunks.
3. Save each chunk as a Blob.
4. Save a List object whose links point to the chunk CIDs in file order.
5. Return the Blob CID or List CID.

This keeps the basic Blob case simple while making List visible in demonstrations.

## Path Resolution

`Resolve(rootCID, path, store)` returns the target CID and object type.

Rules:

- Empty path, `/`, and `.` refer to the root object.
- A path such as `/docs/report.txt` is split into `docs` and `report.txt`.
- Resolution starts from `rootCID`.
- For each path component, the current object must be a Tree.
- The resolver searches the Tree links for a matching `Name`.
- If found, it moves to that child CID.
- If not found, it returns a path-not-found error.
- If a non-Tree object is reached before all components are consumed, it returns a not-a-directory error.

Resolve does not expand List objects. It only locates the target object.

## File Reading

`ReadFile(rootCID, path, store)` uses `Resolve` first.

After resolving:

- If the target is Blob, return `Data`.
- If the target is List, read every linked Blob/List in order and concatenate the bytes.
- If the target is Tree, return a target-is-directory/not-file error.

List reading can be implemented as a helper that reads by CID, so future nested Lists would work naturally even though the importer only creates one List level.

## Directory Listing

`List(rootCID, path, store)` resolves the path and requires the target to be a Tree.

For each direct Link:

1. Load the linked object.
2. Print its name, type, CID, and optional size.

This supports a useful demo command and helps explain Tree as a directory object.

## CLI Design

Commands:

```text
mdag add <local-path>
mdag resolve <root-cid> <path>
mdag cat <root-cid> <path>
mdag ls <root-cid> <path>
```

Output examples:

```text
$ mdag add ./testdata/demo
Root CID: 94fd...

$ mdag resolve 94fd... /docs/report.txt
Target CID: a13c...
Type: blob

$ mdag cat 94fd... /docs/report.txt
This is a report.

$ mdag ls 94fd... /docs
blob  report.txt  a13c...
blob  notes.txt   b81e...
```

The CLI should keep formatting plain and predictable because README examples and live demonstration matter more than visual polish.

## Errors

Use clear errors rather than panics.

Expected cases:

- Object file does not exist for a CID.
- Object JSON cannot be decoded.
- Loaded object's recomputed CID does not match the requested CID.
- A path component does not exist in a Tree.
- Path traversal expects a Tree but reaches Blob or List.
- `cat` target is a Tree.
- `ls` target is not a Tree.
- CLI arguments are missing or malformed.

## Tests

Tests should cover the scoring-critical behavior:

- Same object produces same CID.
- Different object content produces different CID.
- Importing the same directory twice produces the same root CID.
- Modifying a file changes the file object CID and ancestor Tree CID.
- Resolving `/docs/report.txt` returns a file object.
- Reading `/docs/report.txt` returns the original content.
- Reading a file larger than 1024 bytes returns original content through List.
- Resolving `/docs/missing.txt` returns a path-not-found error.
- Calling `cat` on `/docs` returns a target-not-file error.
- Tampering with an object file is detected by integrity verification.

`testdata/demo` should include:

```text
demo/
├── README.md
├── docs/
│   ├── report.txt
│   └── notes.txt
├── data/
│   └── sample.txt
└── big/
    └── large.txt
```

`large.txt` should be larger than 1024 bytes to demonstrate List.

## Defense Explanation

The core explanation:

- CID is the content fingerprint of a serialized object.
- Blob stores bytes.
- Tree stores names and child CIDs, so it works like a directory.
- List stores ordered child CIDs, so it works like a chunked large file.
- HashLink means a parent does not embed the child object directly; it stores the child's CID.
- A directory root CID identifies the whole imported directory because every child CID contributes to its parent Tree CID.
- If a file changes, its Blob/List CID changes, then its parent Tree CID changes, and eventually the root CID changes.
- Resolve only finds the target object; cat reads bytes from that object.

## Implementation Order

1. Initialize Go module and object package.
2. Implement deterministic encoding and CID generation.
3. Implement file-backed Store and integrity verification.
4. Implement file/directory importer with Blob, Tree, and List.
5. Implement Resolve, ReadFile, and List.
6. Implement CLI commands.
7. Add testdata, unit tests, and README examples.
8. Run full verification and prepare defense notes.
