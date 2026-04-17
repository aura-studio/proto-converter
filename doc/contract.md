# contract 子包

**包路径**: `converter/core/contract`

**职责**: 集中定义所有核心接口，作为功能层各模块之间的契约。该包仅依赖 `core/model` 和标准库，是整个系统接口定义的唯一来源。

## 文件列表

### contract.go — 核心接口定义

定义 5 个核心接口，所有方法签名使用 `core/model` 包的类型：

| 接口 | 方法 | 默认实现 |
|------|------|----------|
| `Parser` | `ParseFile`, `ScanTopLevelBlocks`, `StripComments`, `ExtractTypeRefs` | `parser.ProtoParser` |
| `TypeResolver` | `BuildIndex`, `Resolve` | `resolver.TypeResolver` |
| `DepResolverIface` | `CollectWithImportsAndRoots` | `resolver.DepResolver` |
| `Formatter` | `Sanitize`, `StripSelfPackageQualifiers`, `TransformFieldNames`, `WriteLangNamespaceOption` | `formatter.OutputFormatter` |
| `DefPruner` | `PruneMessageFields`, `PruneOneofFields`, `CollectTypeTokens` | `pruner.DefinitionPruner` |

**接口方法签名详情**:

#### Parser

| 方法 | 签名 | 说明 |
|------|------|------|
| `ParseFile` | `(path string) (*model.PFile, error)` | 读取并解析一个 .proto 文件 |
| `ScanTopLevelBlocks` | `(src string) []model.Block` | 扫描 proto 源码中所有顶层 message/enum 定义块 |
| `StripComments` | `(src string) string` | 移除行注释和块注释 |
| `ExtractTypeRefs` | `(src string) []string` | 从去注释后的源码中提取字段的类型引用 |

#### TypeResolver

| 方法 | 签名 | 说明 |
|------|------|------|
| `BuildIndex` | `(parsed map[string]*model.PFile) (fullIndex map[string]model.DefRef, simpleIndex map[string][]model.DefRef)` | 构建全限定名索引和简单名索引 |
| `Resolve` | `(curFile, curPkg, token string) (model.DefRef, bool)` | 解析一个类型 token 到具体定义 |

#### DepResolverIface

| 方法 | 签名 | 说明 |
|------|------|------|
| `CollectWithImportsAndRoots` | `(seeds []model.ProtoItem, importDir string) ([]model.ProtoItem, []model.ProtoItem, error)` | 从种子文件出发，递归解析 import 依赖 |

> `DepResolverIface` 使用 `Iface` 后缀是为了避免与同名结构体 `DepResolver` 冲突。

#### Formatter

| 方法 | 签名 | 说明 |
|------|------|------|
| `Sanitize` | `(src string) string` | 对 proto 输出执行完整的清理流程 |
| `StripSelfPackageQualifiers` | `(content, selfPkg string) string` | 移除当前包的冗余限定前缀 |
| `TransformFieldNames` | `(def, caseKind string) string` | 按指定命名风格转换字段名 |
| `WriteLangNamespaceOption` | `(b *strings.Builder, lang, ns string)` | 根据目标语言写入对应的 option 语句 |

#### DefPruner

| 方法 | 签名 | 说明 |
|------|------|------|
| `PruneMessageFields` | `(def string, keepSet map[string]struct{}) string` | 根据 keepSet 裁剪 message 字段 |
| `PruneOneofFields` | `(blk string, keepSet map[string]struct{}) string` | 根据 keepSet 裁剪 oneof 块字段 |
| `CollectTypeTokens` | `(def string) []string` | 从定义文本中收集所有类型 token |

## 依赖

仅依赖 `converter/core/model` 和标准库 `strings`。

## 设计说明

- 将所有接口放在单个文件 `contract.go` 中，因为接口数量适中（5 个），且它们共同构成一个完整的契约集合
- `pruner` 等功能层包直接引用 `contract` 包中的接口，不再本地重复定义
- 该包位于 `core/` 基础层，确保接口定义不依赖任何功能层实现
- `exporter` 编排层通过这些接口实现依赖注入，支持测试时替换任意组件
