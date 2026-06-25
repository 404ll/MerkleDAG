# Merkle DAG 课程项目

本项目是一个使用 Go 语言实现的本地简化版 Merkle DAG 文件存储与路径解析系统。项目重点是演示内容寻址、CID、HashLink、Tree 路径解析和文件读取流程，不涉及真实 IPFS 网络、DHT、Bitswap、IPNS、UnixFS protobuf 等复杂功能。

## 一、功能完成情况

已实现基础功能：

- Blob：保存普通小文件内容。
- Tree：保存目录的直接子项，每个子项包含名称和子对象 CID。
- CID：使用 `hex(SHA256(JSON对象))` 作为对象标识。
- 对象存储：根据 CID 保存和读取对象。
- 完整性复验：`GetObject` 读取对象后会重新计算 CID，检测对象文件是否被修改。
- 目录导入：递归导入本地文件或目录。
- 路径解析：从根 CID 出发，按路径逐层查找目标对象。
- 文件读取：`cat` 可以输出目标文件内容。
- 异常处理：支持 CID 不存在、路径不存在、路径中途遇到非 Tree、cat 目标不是文件等错误。

已实现提高功能：

- List：大于 1024 字节的文件会被切分为多个 Blob，并通过 List 按顺序链接。
- 持久化存储：对象保存到本地 `data/objects/<cid>.json`。
- `ls` 命令：列出 Tree 目录中的直接子项。

## 二、项目结构

```text
cmd/mdag/main.go        命令行入口
object/object.go        Blob、Tree、List、Link 等对象结构
object/codec.go         JSON 序列化、反序列化和 CID 生成
store/store.go          Store 接口
store/file.go           文件持久化对象存储
importer/importer.go    文件/目录递归导入，支持 List 分块
resolver/resolver.go    路径解析、文件读取和目录列出
testdata/demo/          可直接演示的测试目录
```

## 三、编译与测试

编译：

```bash
go build ./cmd/mdag
```

运行全部测试：

```bash
go test ./...
```

测试覆盖了 CID 稳定性、对象存储、完整性复验、目录导入、List 分块、路径解析、文件读取和错误处理。

## 四、命令说明

导入文件或目录：

```bash
./mdag add <local-path>
```

解析路径：

```bash
./mdag resolve <root-cid> <path>
```

读取文件内容：

```bash
./mdag cat <root-cid> <path>
```

列出目录：

```bash
./mdag ls <root-cid> <path>
```

## 五、运行示例

导入演示目录：

```bash
./mdag add ./testdata/demo
```

输出示例：

```text
根 CID: <root-cid>
```

解析两级路径：

```bash
./mdag resolve <root-cid> /docs/report.txt
```

输出示例：

```text
目标 CID: <target-cid>
类型: Blob（文件）
```

读取文件内容：

```bash
./mdag cat <root-cid> /docs/report.txt
```

输出示例：

```text
这是 Merkle DAG 课程项目的报告文件。
```

列出目录：

```bash
./mdag ls <root-cid> /docs
```

输出示例：

```text
Blob（文件）    notes.txt    <cid>    49
Blob（文件）    report.txt   <cid>    49
```

演示 List 分块文件：

```bash
./mdag resolve <root-cid> /big/large.txt
```

输出示例：

```text
目标 CID: <target-cid>
类型: List（分块文件）
```

`testdata/demo/big/large.txt` 大于 1024 字节，因此导入时会被切分为多个 Blob，再由一个 List 对象按顺序链接这些 Blob。

## 六、核心原理

### 1. CID 如何生成

对象会先被序列化为 JSON，然后对 JSON 字节计算 SHA-256，最后转成十六进制字符串作为 CID：

```text
CID = hex(SHA256(JSON(object)))
```

同一个对象的 JSON 内容相同，因此 CID 相同；对象内容发生变化时，JSON 内容变化，CID 也会变化。

### 2. Blob、Tree、List 的区别

- Blob 保存文件字节。
- Tree 表示目录，保存子项名称和子对象 CID。
- List 表示分块文件，按顺序保存多个 Blob 的 CID。

### 3. HashLink 是什么

HashLink 指父对象不直接保存子对象内容，而是保存子对象的 CID。

例如目录 `Tree` 中的一个 Link 会保存：

```text
Name: report.txt
CID:  <report.txt 对应 Blob 的 CID>
```

这样父 Tree 就通过 CID 指向子对象，形成 Merkle DAG。

### 4. 为什么修改文件会导致根 CID 变化

如果文件内容改变，文件对应的 Blob 或 List CID 会改变。父目录 Tree 中保存的子 CID 也会改变，因此父 Tree 的 CID 改变。这个变化会继续向上传播，最终导致根 Tree CID 改变。

### 5. Resolve 和 Cat 的区别

`resolve` 只负责从根 CID 和路径定位目标对象，返回目标 CID 和类型。

`cat` 会先调用路径解析，再读取目标对象内容。如果目标是 Blob，直接返回 Data；如果目标是 List，按 Link 顺序读取所有分块并拼接；如果目标是 Tree，则返回“目标不是文件”的错误。

## 七、答辩演示流程

1. 编译项目：`go build ./cmd/mdag`
2. 导入目录：`./mdag add ./testdata/demo`
3. 解析文件：`./mdag resolve <root-cid> /docs/report.txt`
4. 读取文件：`./mdag cat <root-cid> /docs/report.txt`
5. 列出目录：`./mdag ls <root-cid> /docs`
6. 演示 List：`./mdag resolve <root-cid> /big/large.txt`
7. 修改 `testdata/demo/docs/report.txt` 后重新导入，观察根 CID 变化。

## 八、小组分工

本项目由本人独立完成，负责对象模型、CID 生成、对象存储、目录导入、路径解析、命令行、测试和 README 编写。
