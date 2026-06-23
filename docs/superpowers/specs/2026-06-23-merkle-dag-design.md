# Merkle DAG 课程项目设计文档

## 项目目标

使用 Go 语言实现一个本地运行的简化版 Merkle DAG 文件存储与路径解析系统。

这个项目的重点不是复现完整 IPFS，而是把课程中学习到的内容寻址、CID、HashLink、Tree 路径解析和文件读取流程串起来。最终项目需要能够编译、运行、演示，并且便于在答辩时解释清楚。

本项目会实现必做的 Blob 和 Tree 对象，并加入一个小型 List 扩展，用来表示分块后的大文件。

## 功能范围

必做功能：

- 将普通小文件编码为 Blob 对象。
- 将目录编码为 Tree 对象。
- 使用 `hex(SHA256(serializedObject))` 作为 CID。
- 将对象持久化保存到 `./data/objects/<cid>.json`。
- 使用 `mdag add` 递归导入文件或目录。
- 使用 `mdag resolve` 从根 CID 和路径解析目标对象。
- 使用 `mdag cat` 读取目标文件内容。
- 对 CID 不存在、路径不存在、路径中途遇到非 Tree 对象、cat 目标不是文件、对象反序列化失败等情况给出明确错误。

提高功能：

- 大于 1024 字节的文件编码为 List 对象，List 内部按顺序链接多个 Blob 分块。
- 读取 List 对象时，按 Link 顺序递归读取并拼接数据。
- 增加 `mdag ls`，用于列出 Tree 的直接子项。
- `GetObject` 读取对象后重新计算 CID，用于检测对象文件是否被篡改。

不做的内容：

- 真实 IPFS 网络、DHT、Bitswap、Provider Discovery、IPNS 或 DNSLink。
- 完整 CIDv0/CIDv1、multibase、multicodec 或 multihash 标准。
- UnixFS protobuf、HAMT-Sharded Directory、并发下载、缓存层、图形界面或 HTTP Gateway。

## 项目结构

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

各目录职责：

- `object`：定义对象类型，并实现确定性的 JSON 编码和 CID 生成。
- `store`：负责对象持久化保存、读取和完整性复验。
- `importer`：负责把本地文件或目录转换成 Merkle DAG 对象。
- `resolver`：负责路径解析、文件读取和目录列出。
- `cmd/mdag`：负责命令行参数解析和用户可见输出。

## 对象模型

项目使用一个统一的对象结构：

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

三种对象的含义：

- Blob：保存文件字节数据，内容放在 `Data` 字段中。
- Tree：表示目录，`Links` 中保存当前目录的直接子项。每个 Link 包含子项名称和子对象 CID。
- List：表示分块文件，`Links` 中按顺序保存多个 Blob 分块的 CID。List 的 Link 可以不需要名称，因为读取时主要依赖顺序。

Tree 中的 Link 是从父目录对象指向子对象的 HashLink。List 中的 Link 是从一个大文件对象指向多个 Blob 分块的 HashLink。

## CID 与序列化

CID 生成流程：

1. 将对象序列化为确定性的 JSON。
2. 对序列化后的字节计算 SHA-256。
3. 将哈希结果编码为小写十六进制字符串。

关键规则：同一个逻辑对象必须得到相同的序列化结果和相同的 CID。为了保证重复导入同一目录时根 CID 稳定，Tree 的 Links 会在保存前按名称排序。

## 对象存储

定义 `Store` 接口：

```go
type Store interface {
    PutObject(obj object.Object) (string, error)
    GetObject(cid string) (object.Object, error)
}
```

文件存储会把每个对象保存为：

```text
./data/objects/<cid>.json
```

`PutObject` 的流程：

1. 对对象进行序列化。
2. 根据序列化结果计算 CID。
3. 创建对象存储目录。
4. 将 JSON 内容写入 `./data/objects/<cid>.json`。
5. 返回 CID。

同一个对象重复保存是允许的，并且应该得到同一个 CID。

`GetObject` 的流程：

1. 根据 CID 找到对应 JSON 文件。
2. 读取并反序列化为 Object。
3. 对读出的对象重新计算 CID。
4. 检查重新计算出的 CID 是否等于请求的 CID。
5. 如果不一致，返回完整性错误。

这样可以演示“CID 是内容指纹”：对象文件被手动修改后，它的内容和文件名中的 CID 就对不上了。

## 导入流程

`AddPath(localPath, store)` 返回导入后的根 CID。

导入目录时：

1. 读取当前目录的直接子项。
2. 对每个子项递归调用 `AddPath`。
3. 为每个子项创建一个 Tree Link，记录子项名称、子对象 CID 和必要的大小信息。
4. 按名称排序 Links，保证结果稳定。
5. 保存 Tree 对象并返回它的 CID。

导入文件时：

1. 如果文件大小小于或等于 1024 字节，直接读取全部内容并保存为一个 Blob。
2. 如果文件大小大于 1024 字节，将文件按 1024 字节切分。
3. 每个分块保存为一个 Blob。
4. 创建一个 List 对象，Links 按文件顺序指向这些 Blob 分块。
5. 返回 Blob CID 或 List CID。

这样既保留了基础版本中“文件就是 Blob”的简单理解，也可以展示 List 如何表示一个由多个 Blob 组成的大文件。

## 路径解析

`Resolve(rootCID, path, store)` 返回目标对象的 CID 和类型。

解析规则：

- 空路径、`/` 和 `.` 都表示根对象。
- `/docs/report.txt` 会被拆分为 `docs` 和 `report.txt` 两个路径片段。
- 解析从 `rootCID` 对应的对象开始。
- 每处理一个路径片段，当前对象都必须是 Tree。
- Resolver 在当前 Tree 的 Links 中查找同名子项。
- 如果找到，就移动到该子项 CID。
- 如果找不到，就返回路径不存在错误。
- 如果路径尚未解析完，但当前对象已经不是 Tree，就返回“当前对象不是目录”的错误。

Resolve 不展开 List。Resolve 的职责只是定位目标对象，真正读取字节内容由 `cat` 或 `ReadFile` 完成。

## 文件读取

`ReadFile(rootCID, path, store)` 会先调用 Resolve 找到目标对象。

解析完成后：

- 如果目标是 Blob，返回 `Data`。
- 如果目标是 List，按 Link 顺序读取每个 Blob/List，并拼接字节。
- 如果目标是 Tree，返回“目标不是文件”的错误。

List 的读取可以实现成一个按 CID 读取内容的辅助函数。这样即使未来出现 List 嵌套 List，也能自然支持。当前 importer 只需要创建一层 List。

## 目录列出

`List(rootCID, path, store)` 会先解析路径，并要求目标对象必须是 Tree。

对 Tree 中的每个直接 Link：

1. 加载 Link 指向的对象。
2. 输出名称、对象类型、CID 和可选大小。

这个命令可以帮助演示 Tree 就像一个目录表：它本身不存储子对象内容，只保存子对象的名称和 CID。

## 命令行设计

命令：

```text
mdag add <local-path>
mdag resolve <root-cid> <path>
mdag cat <root-cid> <path>
mdag ls <root-cid> <path>
```

输出示例：

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

命令行输出保持简单、稳定、容易复制。这个项目的重点是 README 示例和现场演示，不需要复杂的终端界面。

## 异常处理

项目不需要复杂错误体系，但不能 panic 或无提示崩溃。

需要处理的常见错误：

- CID 对应的对象文件不存在。
- 对象 JSON 无法反序列化。
- 读取对象后重新计算的 CID 与请求 CID 不一致。
- Tree 中找不到某个路径片段。
- 路径还没解析完，但当前对象已经是 Blob 或 List。
- `cat` 的目标是 Tree。
- `ls` 的目标不是 Tree。
- CLI 参数缺失或格式不正确。

## 测试设计

测试应覆盖评分中最关键的行为：

- 相同对象得到相同 CID。
- 对象内容变化后得到不同 CID。
- 同一目录重复导入两次，根 CID 相同。
- 修改文件后重新导入，文件对象 CID 和祖先 Tree CID 都变化。
- 解析 `/docs/report.txt` 能返回文件对象。
- 读取 `/docs/report.txt` 能返回原始文件内容。
- 读取大于 1024 字节的文件时，能通过 List 拼接还原原始内容。
- 解析 `/docs/missing.txt` 返回路径不存在错误。
- 对 `/docs` 执行 `cat` 返回目标不是文件错误。
- 手动篡改对象文件后，`GetObject` 能检测到完整性错误。

`testdata/demo` 建议包含：

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

其中 `large.txt` 应该大于 1024 字节，用来稳定演示 List。

## 答辩解释要点

核心解释：

- CID 是序列化对象内容的指纹。
- Blob 保存文件字节。
- Tree 保存名称和子对象 CID，所以它相当于目录。
- List 保存有顺序的子对象 CID，所以它相当于一个分块大文件。
- HashLink 的意思是父对象不直接嵌入子对象内容，而是保存子对象的 CID。
- 一个目录的根 CID 可以代表整个目录，因为每个子对象 CID 都会影响父 Tree 的 CID，最终影响根 Tree 的 CID。
- 如果某个文件内容变化，它的 Blob 或 List CID 会变化，父 Tree CID 也会变化，最后根 CID 也会变化。
- Resolve 只负责定位目标对象；cat 才负责读取对象字节。

老师可能问：

- 为什么相同文件导入两次 CID 一样？
  - 因为对象序列化内容一样，SHA-256 的输入一样，所以 CID 一样。
- 为什么修改一个文件后根 CID 也会变化？
  - 因为文件对象 CID 变化，父目录 Tree 中保存的子 CID 变化，父 Tree 的序列化内容变化，CID 继续向上传播。
- Tree 和 List 有什么区别？
  - Tree 用名称查找子对象，表示目录；List 按顺序读取子对象，表示分块文件。
- Resolve 和 cat 有什么区别？
  - Resolve 找到目标对象 CID；cat 在 Resolve 的基础上读取 Blob 或 List 的字节内容。

## 实现顺序

1. 初始化 Go module 和 `object` 包。
2. 实现对象结构、确定性编码和 CID 生成。
3. 实现文件存储 Store 和完整性复验。
4. 实现文件/目录 importer，支持 Blob、Tree 和 List。
5. 实现 Resolve、ReadFile 和 List。
6. 实现 CLI 命令。
7. 添加 testdata、单元测试和 README 示例。
8. 运行完整验证，并整理答辩说明。
