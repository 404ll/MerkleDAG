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

// AddPath 将本地文件或目录添加到 Merkle DAG 中，并返回根对象的 CID。
func AddPath(localPath string, st store.Store) (string, error) {
	info, err := os.Stat(localPath) // 读取本地文件或文件夹的元数据
	if err != nil {
		return "", err
	}
	//文件、文件夹分类处理
	if info.IsDir() {
		return addDirectory(localPath, st)
	}
	return addFile(localPath, info.Size(), st)
}

// addDirectory 递归导入目录内容，按名称排序链接后生成目录对象。
func addDirectory(localPath string, st store.Store) (string, error) {
	entries, err := os.ReadDir(localPath)
	if err != nil {
		return "", err
	}

	links := make([]object.Link, 0, len(entries))
	//按照路径遍历
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
		//将子对象的链接信息添加到目录对象的链接列表中
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

// addFile 根据文件大小决定直接存为 Blob，或拆成分块列表对象。
func addFile(localPath string, size int64, st store.Store) (string, error) {
	//判断是否需要分块处理
	if size <= ChunkSize {
		data, err := os.ReadFile(localPath) //读取文件内容
		if err != nil {
			return "", err
		}
		//小文件直接处理为Blob
		return st.PutObject(object.Object{
			Type: object.BlobType,
			Data: data,
		})
	}
	//大文件分块处理
	return addChunkedFile(localPath, st)
}

// addChunkedFile 将大文件按固定大小分块，每块存为 Blob，再生成 List 对象串联这些块。
func addChunkedFile(localPath string, st store.Store) (string, error) {
	file, err := os.Open(localPath) //打开文件夹
	if err != nil {
		return "", err
	}
	defer file.Close()

	var links []object.Link
	buffer := make([]byte, ChunkSize)

	for {
		n, readErr := file.Read(buffer)

		if n > 0 {
			//将读取的字节切片复制到一个新的切片中，以避免在下一次读取时覆盖数据
			chunk := make([]byte, n)

			//复制读取的字节到新的切片
			copy(chunk, buffer[:n]) //截取不足的部分

			chunkCID, err := st.PutObject(object.Object{
				Type: object.BlobType,
				Data: chunk,
			})
			if err != nil {
				return "", err
			}
			//将块的CID和大小添加到链接列表中
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

	//将所有块的链接作为一个列表对象存储，并返回其CID
	return st.PutObject(object.Object{
		Type:  object.ListType,
		Links: links,
	})
}
