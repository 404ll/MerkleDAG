package object

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
