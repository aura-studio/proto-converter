# exporter 编排层

**包路径**: `converter/exporter`

**职责**: 作为项目的编排层，导入所有子包并通过 `Exporter` 编排整个裁剪导出流程。是 `main.go` 唯一直接依赖的 converter 子包。

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
| `parser` | `contract.Parser` | 可选注入的解析器 |
| `typeResolver` | `contract.TypeResolver` | 可选注入的类型解析器 |
| `formatter` | `contract.Formatter` | 可选注入的格式化器 |
| `defPruner` | `contract.DefPruner` | 可选注入的裁剪器 |
| `depResolver` | `contract.DepResolverIface` | 可选注入的依赖解析器 |

**构造函数与选项**:

| 函数 | 说明 |
|------|------|
| `NewExporter(opts ...ExporterOption) *Exporter` | 创建 Exporter，支持 functional options |
| `WithParser(p contract.Parser) ExporterOption` | 注入自定义解析器 |
| `WithTypeResolver(r contract.TypeResolver) ExporterOption` | 注入自定义类型解析器 |
| `WithFormatter(f contract.Formatter) ExporterOption` | 注入自定义格式化器 |
| `WithDefPruner(d contract.DefPruner) ExporterOption` | 注入自定义裁剪器 |

**`Run()` 方法流程**:
1. 初始化依赖：未注入时使用默认实现（`parser.ProtoParser`、`resolver.NewTypeResolver()`、`formatter.OutputFormatter`、`pruner.DefinitionPruner`）
2. 加载配置：`config.Loader.Load` → `config.Validator.Validate` → `config.Validator.BuildSeedKeep`
3. 将 `config.SeedItem` 转换为 `model.ProtoItem`
4. 从配置填充 Exporter 字段（ExportDir、ImportDir、Language 等）
5. 校验 Language 合法性（支持 `csharp`/`cs`/`c#`、`golang`/`go`、`lua`）
6. 解析依赖：`resolver.DepResolver.CollectWithImportsAndRoots`
7. 组装 `pruner.Pruner` 和 `model.PruneOptions`
8. 执行裁剪：`pruner.Pruner.BuildPrunedTempProtos`

**依赖注入示例**:

```go
import "github.com/aura-studio/proto-converter/converter/exporter"

exp := exporter.NewExporter(
    exporter.WithParser(mockParser),
    exporter.WithFormatter(mockFormatter),
)
```

**向后兼容**: 支持直接赋值方式：

```go
exp := &exporter.Exporter{}
exp.ConfigPath = configAbs
exp.Run()
```

## 依赖

该包是唯一的组装点，依赖所有子包：

- `converter/core/contract` — 接口类型定义
- `converter/core/model` — 共享数据类型（`ProtoItem`、`PruneOptions`）
- `converter/internal/config` — 配置加载与校验
- `converter/internal/parser` — proto 文件解析（默认实现）
- `converter/internal/resolver` — 类型与依赖解析（默认实现）
- `converter/internal/pruner` — 裁剪编排（默认实现）
- `converter/internal/formatter` — 输出格式化（默认实现）
