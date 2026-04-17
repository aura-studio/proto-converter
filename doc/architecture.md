# 架构设计说明

## 整体架构

项目采用三级嵌套子包架构，按依赖层次将包分为 `core/`（基础层）、`internal/`（功能层）和 `exporter/`（编排层）三组。通过无依赖的叶子包 `core/model` 承载共享数据类型，通过 `core/contract` 集中定义核心接口，避免循环依赖。

```
main.go
  └── converter/exporter/exporter.go（编排层）
        ├── internal/config/       加载 YAML（Viper）→ 校验 → 构建种子数据
        ├── internal/resolver/     解析 import 依赖 → 收集所有可达 proto 文件
        └── internal/pruner/       裁剪编排（使用 core/contract 接口）
              ├── internal/parser/     解析 .proto 文件为结构化表示
              ├── internal/resolver/   构建类型索引 → 解析类型引用
              ├── internal/pruner/     字段级裁剪（defpruner）
              └── internal/formatter/  格式化输出 → 写文件
```

### 三级目录结构

```
converter/
├── core/                  ← 基础层（零业务依赖的共享基础设施）
│   ├── model/             ← 共享数据类型（叶子层，无依赖）
│   ├── util/              ← 共享工具函数（仅依赖标准库）
│   └── contract/          ← 核心接口定义（仅依赖 core/model + 标准库）
├── internal/              ← 功能层（内部实现模块，Go internal 可见性限制）
│   ├── config/            ← 配置加载与校验（使用 Viper）
│   ├── parser/            ← proto 文件解析
│   ├── resolver/          ← 类型与依赖解析
│   ├── pruner/            ← 定义裁剪编排
│   └── formatter/         ← 输出格式化
└── exporter/              ← 编排层（顶层入口，组装所有模块）
```

- **`core/`**：基础层，包含所有子包共享的类型定义（`model`）、工具函数（`util`）和接口契约（`contract`）。这些包没有业务逻辑，是整个系统的地基。
- **`internal/`**：功能层，包含各个具体的业务实现模块。它们依赖 `core/` 中的基础设施，彼此之间仅有有限的依赖（如 `pruner` 依赖 `resolver` 的 `WellKnownTypes`）。Go 语言的 `internal` 目录可见性限制确保这些包不会被 `converter/` 外部直接导入。
- **`exporter/`**：编排层，作为唯一的顶层入口，组装和调度所有功能模块。

## 数据流

```
YAML 配置文件
    │
    ▼
config.Loader.Load()          ─── 使用 Viper 读取并反序列化
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
    ├── parseAndIndex()                            ─── 阶段 1：解析与索引
    │   ├── parser.ProtoParser.ParseFile()         ─── 解析每个 .proto 为 PFile
    │   └── resolver.TypeResolver.BuildIndex()     ─── 构建类型名→定义的索引
    │
    ├── collectSelectedDefs()                      ─── 阶段 2：BFS 依赖追踪
    │   ├── pruner.DefinitionPruner.CollectTypeTokens() ─── 收集类型引用
    │   ├── resolver.TypeResolver.Resolve()             ─── 解析引用
    │   └── pruner.DefinitionPruner.PruneMessageFields() ── 按 keepSet 裁剪字段
    │
    └── assembleAndWrite()                         ─── 阶段 3：组装与输出
        ├── formatter.OutputFormatter.StripSelfPackageQualifiers()
        ├── formatter.OutputFormatter.TransformFieldNames()
        ├── formatter.OutputFormatter.WriteLangNamespaceOption()
        └── formatter.OutputFormatter.Sanitize() ─── 最终清理
```

## 包依赖关系

```
core/model（叶子包，无外部依赖）
core/util（仅依赖标准库）
core/contract（仅依赖 core/model + 标准库）
  ▲
  │
  ├── internal/config      （仅依赖标准库 + Viper）
  ├── internal/parser      （依赖 core/model）
  ├── internal/resolver    （依赖 core/model + core/util）
  ├── internal/pruner      （依赖 core/model + core/contract + core/util + internal/resolver）
  ├── internal/formatter   （依赖 core/model）
  │
  └── exporter（编排层，依赖所有子包）
        ▲
        │
        main.go
```

关键设计决策：
- 所有功能子包通过 `core/model` 获取共享类型，通过 `core/contract` 获取接口定义
- `pruner` 依赖 `resolver` 的 `WellKnownTypes` 是功能层内部唯一的直接依赖
- `exporter` 作为唯一的组装点，导入所有子包并通过 `Exporter` 编排整个流程
- `main.go` 只依赖 `exporter` 包
- `core/util` 提供共享工具函数（如 `BaseName`），消除了 `resolver` 和 `pruner` 之间的重复代码

## 接口与依赖注入

`converter/core/contract/contract.go` 集中定义了 5 个核心接口：

| 接口 | 默认实现 | 职责 |
|------|----------|------|
| `Parser` | `parser.ProtoParser` | .proto 文件解析 |
| `TypeResolver` | `resolver.TypeResolver` | 类型引用解析 |
| `DepResolverIface` | `resolver.DepResolver` | import 依赖解析 |
| `Formatter` | `formatter.OutputFormatter` | 输出格式化 |
| `DefPruner` | `pruner.DefinitionPruner` | 定义裁剪 |

`Exporter`（位于 `converter/exporter/exporter.go`）通过 functional options 模式支持依赖注入，便于测试时替换任意组件：

```go
exp := exporter.NewExporter(
    exporter.WithParser(mockParser),
    exporter.WithFormatter(mockFormatter),
)
```
