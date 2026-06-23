package object

import "testing"

func TestCIDStableForSameObject(t *testing.T) {
	obj := Object{Type: BlobType, Data: []byte("hello")}

	cid1, err := CID(obj)
	if err != nil {
		t.Fatal(err)
	}
	cid2, err := CID(obj)
	if err != nil {
		t.Fatal(err)
	}

	if cid1 != cid2 {
		t.Fatalf("相同的对象应该得到相同的CID: %s != %s", cid1, cid2)
	}
}

func TestCIDChangesWhenContentChanges(t *testing.T) {
	first := Object{Type: BlobType, Data: []byte("hello")}
	second := Object{Type: BlobType, Data: []byte("hello!")}

	cid1, err := CID(first)
	if err != nil {
		t.Fatal(err)
	}
	cid2, err := CID(second)
	if err != nil {
		t.Fatal(err)
	}

	if cid1 == cid2 {
		t.Fatalf("不同的对象应该得到不同的CID: %s", cid1)
	}
}

func TestEncodeUsesDeterministicJSON(t *testing.T) {
	obj := Object{
		Type: TreeType,
		Links: []Link{
			{Name: "b.txt", CID: "cid-b", Size: 2},
			{Name: "a.txt", CID: "cid-a", Size: 1},
		},
	}

	first, err := Encode(obj)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Encode(obj)
	if err != nil {
		t.Fatal(err)
	}

	if string(first) != string(second) {
		t.Fatalf("序列化应该是确定的:\n%s\n%s", first, second)
	}
}
