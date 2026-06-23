# Merkle DAG 课程项目

这是一个使用 Go 语言实现的简化版 Merkle DAG 文件存储与路径解析系统。

## 已完成功能

- Blob：保存小文件内容。
- Tree：保存目录的直接子项名称和子对象 CID。
- List：将大于 1024 字节的文件切分为多个 Blob，并按顺序链接。
- CID：使用 `hex(SHA256(JSON对象))` 生成内容标识。
- 本地对象存储：对象保存到 `./data/objects/<cid>.json`。
- 完整性复验：读取对象后重新计算 CID，检测对象文件是否被修改。
- 命令行：支持 `add`、`resolve`、`cat`、`ls`。

## 编译

```bash
go build ./cmd/mdag
```

## 运行示例

导入演示目录：

```bash
./mdag add ./testdata/demo
```

输出示例：

```text
Root CID: <root-cid>
```

解析两级路径：

```bash
./mdag resolve <root-cid> /docs/report.txt
```

读取文件：

```bash
./mdag cat <root-cid> /docs/report.txt
```

列出目录：

```bash
./mdag ls <root-cid> /docs
```

演示 List：

```bash
./mdag resolve <root-cid> /big/large.txt
./mdag cat <root-cid> /big/large.txt
```

`large.txt` 大于 1024 字节，导入后会被编码为 List，List 中的 Link 按顺序指向多个 Blob 分块。

## 核心原理

CID 是对象内容的哈希指纹。同一个对象序列化结果相同，因此 CID 相同；对象内容发生变化，CID 也会变化。

Blob 保存文件字节。Tree 表示目录，保存子项名称和子对象 CID。List 表示分块文件，按顺序保存多个 Blob 的 CID。

HashLink 指父对象不直接嵌入子对象内容，而是保存子对象 CID。目录的根 CID 能代表整个目录，是因为每个子对象 CID 都会影响父 Tree 的 CID，并最终影响根 Tree 的 CID。

Resolve 只负责根据路径找到目标对象 CID。Cat 会在 Resolve 的基础上读取 Blob 或 List 的字节内容。

## 测试

```bash
go test ./...
```

## 答辩演示流程

1. 运行 `./mdag add ./testdata/demo`，生成根 CID。
2. 运行 `./mdag resolve <root-cid> /docs/report.txt`，展示路径解析。
3. 运行 `./mdag cat <root-cid> /docs/report.txt`，展示文件读取。
4. 运行 `./mdag resolve <root-cid> /big/large.txt`，展示 List 类型。
5. 修改 `testdata/demo/docs/report.txt` 后重新 add，展示根 CID 变化。

## 小组分工

如为个人完成，可写：本项目由本人独立完成，负责对象模型、CID 生成、对象存储、目录导入、路径解析、命令行和测试。
