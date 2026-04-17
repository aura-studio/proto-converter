# pruner 子包

**包路径**: `converter/internal/pruner`

**职责**: 裁剪编排和字段级裁剪。`Pruner` 结构体通过 `core/contract` 包的接口组合 Parser、TypeResolver、DefPruner、Formatter 四个组件，编排整个裁剪流程；`DefinitionPruner` 负责 message/enum 定义的字段级裁剪。

> 注意：该包位于 `internal/` 目录下，受 Go 语言 `internal` 可见性限制，仅允许 `converter/` 内部的包导入。

## 文件列表

### pruner.go — 裁剪编排器

**`Pruner` 结构体**（使用 `core/contract` 包的接口，不再本地定义重复接口）:

| 字段 | 类型 | 说明 |
|------|------|------|
| `Parser` | `contract.Parser` | proto 文件解析器 |
| `Resolver` | `contract.TypeResolver` | 类型引用解析器 |
| `DefPrune` | `contract.DefPruner` | 定义裁剪器 |
| `Fmt` | `contract.Formatter` | 输出格式化器 |

**依赖**: `converter/core/contract`、`converter/core/model`、`converter/core/util`、`converter/internal/resolver`（WellKnownTypes）、标准库。

| 方法 | 说明 |
|------|------|
| `BuildPrunedTempProtos(all, seeds, seedKeep, typeFieldKeep, opts) (outDir, targets, error)` | 裁剪并写出 proto 文件（编排方法，依次调用三个子方法） |

**BuildPrunedTempProtos 已拆分为三个阶段性子方法**:

#### 阶段 1：parseAndIndex — 解析与索引构建

```go
func (p Pruner) parseAndIndex(all []model.ProtoItem) (parseResult, error)
```

- 遍历所有 ProtoItem，调用 `Parser.ParseFile` 解析每个文件
- 调用 `Resolver.BuildIndex` 构建全名索引和简名索引
- 返回 `parseResult` 内部结构体（封装 `parsed`、`fullIndex`、`simpleIndex`）

#### 阶段 2：collectSelectedDefs — BFS 依赖追踪

```go
func (p Pruner) collectSelectedDefs(pr, seeds, seedKeep, typeFieldKeep, inDir) (map[string]map[string]struct{}, error)
```

- 从种子文件的定义出发，根据 seedKeep 规则选择初始定义集
- BFS 遍历选中的定义，对每个定义：
  - 按 typeFieldKeep 规则裁剪字段（`DefPrune.PruneMessageFields`）
  - 收集类型引用（`DefPrune.CollectTypeTokens`）
  - 解析引用到具体定义（`Resolver.Resolve`），加入队列
- 返回 `selected` 映射（文件路径 → 定义名称集合）

#### 阶段 3：assembleAndWrite — 文件组装与输出

```go
func (p Pruner) assembleAndWrite(pr, selected, typeFieldKeep, opts) (string, []model.ProtoItem, error)
```

- 遍历每个解析后的文件，根据 `selected` 映射决定输出内容
- 对选中的定义执行字段裁剪、自包前缀移除（`Fmt.StripSelfPackageQualifiers`）、字段名转换（`Fmt.TransformFieldNames`）
- 计算跨文件 import 和 Google import
- 组装 syntax、package、import、namespace option
- 调用 `Fmt.Sanitize` 清理输出
- 写入目标文件

**编排方法**（≤30 行）:

```go
func (p Pruner) BuildPrunedTempProtos(...) (string, []model.ProtoItem, error) {
    pr, err := p.parseAndIndex(all)
    // ...
    selected, err := p.collectSelectedDefs(pr, seeds, seedKeep, typeFieldKeep, opts.InDir)
    // ...
    return p.assembleAndWrite(pr, selected, typeFieldKeep, opts)
}
```

**`parseResult` 内部结构体**: 封装解析阶段的输出，包含 `parsed`（已解析文件映射）、`fullIndex`（全限定名索引）、`simpleIndex`（简单名索引）。

### defpruner.go — 定义裁剪器

**`DefinitionPruner` 结构体**（空结构体，实现 `contract.DefPruner` 接口）:

| 方法 | 说明 |
|------|------|
| `PruneMessageFields(def string, keepSet map[string]struct{}) string` | 根据 keepSet 裁剪 message 字段 |
| `PruneOneofFields(blk string, keepSet map[string]struct{}) string` | 根据 keepSet 裁剪 oneof 块字段 |
| `CollectTypeTokens(def string) []string` | 从定义文本中收集所有类型 token |

**PruneMessageFields 算法**:
- 定位 message 的 `{...}` 范围
- 逐语句扫描 body，跟踪花括号深度
- 遇到 `oneof` 块：委托 `pruneOneofFields` 处理
- 遇到 `message`/`enum`/`extend` 嵌套块：无条件保留
- 普通字段语句：通过 `keepFieldStmt` 判断是否保留
- 最后清理空行

**PruneOneofFields 算法**:
- 按行扫描 oneof 块
- keepSet 为空时保留所有字段
- 否则仅保留 keepSet 中的字段
- 如果没有任何字段被保留，返回空字符串（整个 oneof 块被移除）

**辅助函数**:

| 函数 | 说明 |
|------|------|
| `ResolveTypeKeepSet(m, pkg, name)` | 按类型名和包名查找对应的 keepSet，先查简单名再查全限定名 |
| `ExtractOriginalBlock(srcData, defText)` | 提取原始块文本（当前直接返回 defText） |

### helpers.go — 裁剪辅助函数

**正则表达式**:

| 变量 | 说明 |
|------|------|
| `reLineComment` | 匹配行注释 `//...` |
| `reBlockComment` | 匹配块注释 `/*...*/` |
| `reMapType` | 匹配 `map<K, V>` 类型声明 |
| `reFieldType` | 匹配字段声明中的类型名 |

**辅助函数**:

| 函数 | 说明 |
|------|------|
| `keepFieldStmt(stmt, keepSet) bool` | 判断字段语句是否应保留。块关键字开头的语句、非分号结尾的语句、keepSet 为空时均保留 |
| `lastIdent(s string) string` | 提取字符串末尾的标识符（用于从 `Type fieldName` 中提取字段名） |
| `looksLikeBlock(s string) bool` | 判断文本是否以 oneof/message/enum/extend 开头 |
| `readKeyword(s, i) (kw, next)` | 从位置 i 读取一个关键字 |
| `readIdentAfter(s, i) (ident, next)` | 从位置 i 读取一个标识符 |
| `findBlock(s, i) (start, end)` | 从位置 i 查找匹配的花括号块，正确处理字符串和注释 |
