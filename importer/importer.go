package importer

import (
	"io"
	"os"
	"path/filepath"
	"sort"

	"merkledag/object"
	"merkledag/store"
)

const ChunkSize = 1024

func AddPath(localPath string, st store.Store) (string, error) {
	info, err := os.Stat(localPath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return addDirectory(localPath, st)
	}
	return addFile(localPath, info.Size(), st)
}

func addDirectory(localPath string, st store.Store) (string, error) {
	entries, err := os.ReadDir(localPath)
	if err != nil {
		return "", err
	}

	links := make([]object.Link, 0, len(entries))
	for _, entry := range entries {
		childPath := filepath.Join(localPath, entry.Name())
		childInfo, err := os.Stat(childPath)
		if err != nil {
			return "", err
		}
		childCID, err := AddPath(childPath, st)
		if err != nil {
			return "", err
		}
		links = append(links, object.Link{
			Name: entry.Name(),
			CID:  childCID,
			Size: childInfo.Size(),
		})
	}

	sort.Slice(links, func(i, j int) bool {
		return links[i].Name < links[j].Name
	})

	return st.PutObject(object.Object{
		Type:  object.TreeType,
		Links: links,
	})
}

func addFile(localPath string, size int64, st store.Store) (string, error) {
	if size <= ChunkSize {
		data, err := os.ReadFile(localPath)
		if err != nil {
			return "", err
		}
		return st.PutObject(object.Object{
			Type: object.BlobType,
			Data: data,
		})
	}
	return addChunkedFile(localPath, st)
}

func addChunkedFile(localPath string, st store.Store) (string, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var links []object.Link
	buffer := make([]byte, ChunkSize)
	for {
		n, readErr := file.Read(buffer)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buffer[:n])
			chunkCID, err := st.PutObject(object.Object{
				Type: object.BlobType,
				Data: chunk,
			})
			if err != nil {
				return "", err
			}
			links = append(links, object.Link{
				CID:  chunkCID,
				Size: int64(n),
			})
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}

	return st.PutObject(object.Object{
		Type:  object.ListType,
		Links: links,
	})
}
