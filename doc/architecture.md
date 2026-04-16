# 架构设计说明

## 整体架构

项目采用分层子包架构，通过一个无依赖的叶子包 `model` 承载共享数据类型，避免循环依赖。

```
main.go
  └── converter/exporter.go（编排层）
        ├── config/       加载 YAML → 校验 → 构建种子数据
        ├── resolver/     解析 import 依赖 → 收集所有可达 proto 文件
        └── pruner/       裁剪编排
              ├── parser/     解析 .proto 文件为结构化表示
              ├── resolver/   构建类型索引 → 解析类型引用
              ├── pruner/     字段级裁剪（defpruner）
              └── formatter/  格式化输出 → 写文件
```

## 数据流

```
YAML 配置文件
    │
    ▼
config.Loader.Load()          ─── 读取并反序列化
    │
    ▼
config.Validator.Validate()   ─── 校验配置完整性
    │
    ▼
config.Validator.BuildSeedKeep() ─── 构建种子列表和保留规则
    │
    ▼
resolver.DepResolver.CollectWithImportsAndRoots()
    │                          ─── 从种子出发，递归解析 import，
    │                              收集所有可达的 .proto 文件
    ▼
pruner.Pruner.BuildPrunedTempProtos()
    │
    ├── parser.ProtoParser.ParseFile()     ─── 解析每个 .proto 为 PFile
    ├── resolver.TypeResolver.BuildIndex() ─── 构建类型名→定义的索引
    ├── 从种子定义出发，BFS 追踪类型依赖
    │   ├── pruner.DefinitionPruner.CollectTypeTokens() ─── 收集类型引用
    │   ├── resolver.TypeResolver.Resolve()             ─── 解析引用
    │   └── pruner.DefinitionPruner.PruneMessageFields() ── 按 keepSet 裁剪字段
    │
    └── 对每个文件组装输出
        ├── formatter.OutputFormatter.StripSelfPackageQualifiers()
        ├── formatter.OutputFormatter.TransformFieldNames()
        ├── formatter.OutputFormatter.WriteLangNamespaceOption()
        └── formatter.OutputFormatter.Sanitize() ─── 最终清理
```

## 包依赖关系

```
model（叶子包，无外部依赖）
  ▲
  │
  ├── config      （仅依赖标准库 + yaml.v3）
  ├── seedloader  （依赖 model）
  ├── parser      （依赖 model）
  ├── resolver    （依赖 model）
  ├── pruner      （依赖 model + resolver）
  ├── formatter   （依赖 model）
  │
  └── converter（顶层包，依赖所有子包）
        ▲
        │
        main.go
```

关键设计决策：所有子包只依赖 `model` 包获取共享类型，不互相依赖（`pruner` 依赖 `resolver` 的 `WellKnownTypes` 是唯一例外）。`converter` 顶层包作为唯一的组装点，导入所有子包并通过 `Exporter` 编排整个流程。

## 接口与依赖注入

`converter/interfaces.go` 定义了 6 个核心接口：

| 接口 | 默认实现 | 职责 |
|------|----------|------|
| `Parser` | `parser.ProtoParser` | .proto 文件解析 |
| `TypeResolver` | `resolver.TypeResolver` | 类型引用解析 |
| `DepResolverIface` | `resolver.DepResolver` | import 依赖解析 |
| `Formatter` | `formatter.OutputFormatter` | 输出格式化 |
| `DefPruner` | `pruner.DefinitionPruner` | 定义裁剪 |
| `SeedLoaderIface` | `seedloader.SeedLoader` | 种子文件规范化 |

`Exporter` 通过 functional options 模式支持依赖注入，便于测试时替换任意组件：

```go
exp := converter.NewExporter(
    converter.WithParser(mockParser),
    converter.WithFormatter(mockFormatter),
)
```
