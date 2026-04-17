# resolver 子包

**包路径**: `converter/internal/resolver`

**职责**: 处理两种解析任务：(1) 类型引用解析 — 将 proto 定义中的类型名解析到具体的定义位置；(2) import 依赖解析 — 从种子文件出发递归收集所有可达的 proto 文件。

> 注意：该包位于 `internal/` 目录下，受 Go 语言 `internal` 可见性限制，仅允许 `converter/` 内部的包导入。实现 `core/contract.TypeResolver` 和 `core/contract.DepResolverIface` 接口。

## 文件列表

### wellknown.go — 常量表

| 变量 | 说明 |
|------|------|
| `WellKnownTypes` | `map[string]string`，Google 官方 proto 类型到 import 路径的映射。包含 Timestamp、Duration、Any、Empty、Struct、Value、ListValue 以及所有 Wrapper 类型 |
| `ScalarTypes` | `map[string]struct{}`，proto 标量类型集合（double、float、int32、int64、string、bool、bytes 等 15 种） |

这两个表被 `TypeResolver.Resolve` 用于快速跳过不需要解析的类型。

### type.go — 类型引用解析器

**`TypeResolver` 结构体**（有状态，持有索引缓存）:

| 字段 | 类型 | 说明 |
|------|------|------|
| `parsed` | `map[string]*model.PFile` | 所有已解析的 proto 文件 |
| `fullIndex` | `map[string]model.DefRef` | 全限定名（`pkg.TypeName`）→ 定义引用 |
| `simpleIndex` | `map[string][]model.DefRef` | 简单名（`TypeName`）→ 定义引用列表 |
| `pkgs` | `map[string]struct{}` | 所有已知的包名集合 |

| 方法 | 说明 |
|------|------|
| `NewTypeResolver() *TypeResolver` | 创建实例 |
| `BuildIndex(parsed) (fullIndex, simpleIndex)` | 遍历所有 PFile，构建全限定名索引和简单名索引 |
| `Resolve(curFile, curPkg, token) (DefRef, bool)` | 解析一个类型 token 到具体定义 |

**Resolve 解析优先级**:
1. 跳过标量类型和 WellKnownTypes
2. 使用 `util.BaseName` 提取简单名，在当前文件内按简单名查找
3. 尝试全限定名解析（`resolveTop`）：
   - 单段名：加当前包前缀查 fullIndex
   - 多段名：首段作为包名查 fullIndex
4. 在 simpleIndex 中查找，仅当唯一匹配时返回

**依赖**: `converter/core/model`、`converter/core/util`、标准库。

**`BaseName` 委托函数**: 保留了导出的 `BaseName` 函数作为向后兼容的委托，内部调用 `util.BaseName`。标记为 `Deprecated`，建议直接使用 `converter/core/util.BaseName`。

> `BaseName` 的规范实现已迁移到 `converter/core/util/util.go`，从可能带包名前缀的 token 中提取简单名（如 `pkg.Foo` → `Foo`）。

### dep.go — import 依赖解析器

**`DepResolver` 结构体**（空结构体）:

| 方法 | 说明 |
|------|------|
| `CollectWithImportsAndRoots(seeds, importDir) (all, resolvedSeeds, error)` | 从种子文件出发，递归解析 import 依赖，返回所有可达的 proto 文件 |

**算法流程**:
1. 收集搜索根目录：种子文件目录 + importDir 下所有子目录 + 工作目录下所有子目录
2. 对每个种子文件，尝试在搜索根中定位实际路径
3. BFS 遍历：读取每个文件的 `import "..."` 语句，在搜索根中查找被导入的文件
4. 按 basename 去重，最终按字母序排序返回

**`importRe` 正则**: 匹配 proto 文件中的 `import "path/to/file.proto";` 语句。
