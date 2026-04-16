# converter 顶层包

**包路径**: `converter`

**职责**: 作为项目的组装层，导入所有子包并通过 `Exporter` 编排整个裁剪导出流程。同时定义核心接口、类型别名和命名风格转换函数。

## 文件列表

### exporter.go — 导出流程编排

**`Exporter` 结构体**:

| 公开字段 | 类型 | 说明 |
|----------|------|------|
| `ConfigPath` | `string` | YAML 配置文件路径 |
| `ExportDir` | `string` | 输出目录 |
| `ImportDir` | `string` | 源码根目录 |
| `Namespace` | `string` | 命名空间 |
| `Language` | `string` | 目标语言 |
| `FileNameCase` | `string` | 文件名命名风格 |
| `FieldNameCase` | `string` | 字段名命名风格 |
| `Prune` | `bool` | 是否裁剪 |
| `DryRun` | `bool` | 是否演练模式 |

| 私有字段 | 类型 | 说明 |
|----------|------|------|
| `parser` | `Parser` | 可选注入的解析器 |
| `typeResolver` | `TypeResolver` | 可选注入的类型解析器 |
| `formatter` | `Formatter` | 可选注入的格式化器 |
| `defPruner` | `DefPruner` | 可选注入的裁剪器 |

**构造函数与选项**:

| 函数 | 说明 |
|------|------|
| `NewExporter(opts ...ExporterOption) *Exporter` | 创建 Exporter，支持 functional options |
| `WithParser(p Parser) ExporterOption` | 注入自定义解析器 |
| `WithTypeResolver(r TypeResolver) ExporterOption` | 注入自定义类型解析器 |
| `WithFormatter(f Formatter) ExporterOption` | 注入自定义格式化器 |
| `WithDefPruner(d DefPruner) ExporterOption` | 注入自定义裁剪器 |

**`Run()` 方法流程**:
1. 初始化依赖：未注入时使用默认实现（`parser.ProtoParser`、`resolver.NewTypeResolver()`、`formatter.OutputFormatter`、`pruner.DefinitionPruner`）
2. 加载配置：`config.Loader.Load` → `config.Validator.Validate` → `config.Validator.BuildSeedKeep`
3. 将 `config.SeedItem` 转换为 `model.ProtoItem`
4. 从配置填充 Exporter 字段（ExportDir、ImportDir、Language 等）
5. 校验 Language 合法性
6. 解析依赖：`resolver.DepResolver.CollectWithImportsAndRoots`
7. 组装 `pruner.Pruner` 和 `model.PruneOptions`
8. 执行裁剪：`pruner.Pruner.BuildPrunedTempProtos`

**向后兼容**: 支持原有的直接赋值方式：
```go
exp := &converter.Exporter{}
exp.ConfigPath = configAbs
exp.Run()
```

### interfaces.go — 核心接口定义

定义 6 个核心接口，所有方法签名使用 `model` 包的类型：

| 接口 | 方法 | 默认实现 |
|------|------|----------|
| `Parser` | `ParseFile`, `ScanTopLevelBlocks`, `StripComments`, `ExtractTypeRefs` | `parser.ProtoParser` |
| `TypeResolver` | `BuildIndex`, `Resolve` | `resolver.TypeResolver` |
| `DepResolverIface` | `CollectWithImportsAndRoots` | `resolver.DepResolver` |
| `Formatter` | `Sanitize`, `StripSelfPackageQualifiers`, `TransformFieldNames`, `WriteLangNamespaceOption` | `formatter.OutputFormatter` |
| `DefPruner` | `PruneMessageFields`, `PruneOneofFields`, `CollectTypeTokens` | `pruner.DefinitionPruner` |
| `SeedLoaderIface` | `SeedsFromList` | `seedloader.SeedLoader` |

> `DepResolverIface` 和 `SeedLoaderIface` 使用 `Iface` 后缀是为了避免与同名结构体冲突。

### types.go — 类型别名

通过 Go 类型别名从 `model` 包重导出所有共享类型，使外部代码可以直接使用 `converter.ProtoItem` 等类型名：

```go
type ProtoItem = model.ProtoItem
type PFile = model.PFile
type TopDef = model.TopDef
type DefRef = model.DefRef
type Block = model.Block
type PruneOptions = model.PruneOptions
```

同时重导出工具函数：`NormalizeItem`、`Exists`、`ShortPath`、`TrimExt`、`EnsureDir`、`BlockBody`。

### case.go — 命名风格转换

| 函数 | 说明 |
|------|------|
| `ToCase(s, caseKind string) string` | 委托 `model.ToCase` 执行转换 |

文件中保留了 `toCamel`、`toSnake` 等内部函数的副本，供 `case_test.go` 测试使用。

### case_test.go — 命名风格单元测试

测试 `toCamel` 和 `toSnake` 函数的各种输入场景，确保命名转换逻辑正确。
