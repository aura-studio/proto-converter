# 流水线输入/输出说明

本文档按数据流顺序，逐环节说明每个模块的输入和输出。

---

## 总览

```
YAML 文件
  │
  ▼
┌─────────┐     ┌───────────┐     ┌──────────┐     ┌─────────┐     ┌───────────┐
│ config  │ ──▶ │ resolver  │ ──▶ │  parser  │ ──▶ │ pruner  │ ──▶ │ formatter │
│ 加载配置 │     │ 解析依赖   │     │ 解析文件  │     │ 裁剪定义  │     │ 格式化输出  │
└─────────┘     └───────────┘     └──────────┘     └─────────┘     └───────────┘
  │                │                 │                │                │
  ▼                ▼                 ▼                ▼                ▼
Config           []ProtoItem       map[path]*PFile  selected defs    .proto 文件
seedKeep         resolvedSeeds     fullIndex        pruned text      (磁盘)
typeFieldKeep                      simpleIndex
```

---

## 环节 1：config — 加载配置

**模块**: `converter/internal/config`

### 输入

| 参数 | 类型 | 说明 |
|------|------|------|
| `path` | `string` | YAML 配置文件路径 |

### 输出

**Loader.Load** 返回：

| 返回值 | 类型 | 说明 |
|--------|------|------|
| `cfg` | `Config` | 反序列化后的配置结构体 |

```go
type Config struct {
    DryRun *bool
    Import ImportSection  // Dir, Prune, Keep{Files, Types}
    Export ExportSection  // Dir, Language, Namespace, FileNameCase, FieldNameCase
}
```

**Validator.BuildSeedKeep** 返回：

| 返回值 | 类型 | 说明 |
|--------|------|------|
| `seeds` | `[]SeedItem` | 规范化后的种子文件列表（Path/Dir/Base） |
| `seedKeep` | `map[string]map[string]struct{}` | 文件级保留规则：文件路径 → 要保留的定义名集合 |
| `typeFieldKeep` | `map[string]map[string]struct{}` | 类型级保留规则：类型名 → 要保留的字段名集合 |

### 示例

YAML 配置：
```yaml
import:
  dir: proto_src
  keep:
    files:
      - file: player.proto
        keep: [PlayerInfo, LoginRequest]
    types:
      - type: PlayerInfo
        keep: [name, level]
```

输出：
```
seeds:       [{Path:"player.proto", Dir:"", Base:"player.proto"}]
seedKeep:    {"player.proto": {"PlayerInfo":{}, "LoginRequest":{}}}
typeFieldKeep: {"PlayerInfo": {"name":{}, "level":{}}}
```

---

## 环节 2：resolver（依赖解析）— 收集所有相关文件

**模块**: `converter/internal/resolver`（`DepResolver`）

### 输入

| 参数 | 类型 | 说明 |
|------|------|------|
| `seeds` | `[]model.ProtoItem` | 种子文件列表（从 config 转换而来） |
| `importDir` | `string` | 源码根目录 |

### 输出

| 返回值 | 类型 | 说明 |
|--------|------|------|
| `all` | `[]model.ProtoItem` | 所有可达的 proto 文件（种子 + 递归 import 的传递闭包），按 basename 排序 |
| `resolvedSeeds` | `[]model.ProtoItem` | 种子文件的实际路径（可能经过搜索根定位修正） |

### 算法

1. 构建搜索根目录列表（种子目录 + importDir 子目录 + 工作目录子目录）
2. 定位每个种子文件的实际路径
3. BFS 遍历：读取每个文件的 `import "..."` 语句，在搜索根中查找被导入的文件
4. 按 basename 去重，按字母序排序返回

### 示例

```
输入 seeds: [{Path:"player.proto"}]
importDir:  "proto_src"

输出 all: [
  {Path:"proto_src/common.proto", ...},
  {Path:"proto_src/player.proto", ...},
  {Path:"proto_src/item.proto", ...},
]
resolvedSeeds: [{Path:"proto_src/player.proto", ...}]
```

---

## 环节 3：parser — 解析 proto 文件

**模块**: `converter/internal/parser`（`ProtoParser`）

### 输入

| 参数 | 类型 | 说明 |
|------|------|------|
| `path` | `string` | 单个 .proto 文件路径 |

> 在 pruner 的 `parseAndIndex` 阶段，对 `all` 中的每个文件调用一次 `ParseFile`。

### 输出

| 返回值 | 类型 | 说明 |
|--------|------|------|
| `pf` | `*model.PFile` | 解析后的结构化表示 |

```go
type PFile struct {
    Path    string     // 文件路径
    Package string     // package 声明，如 "game.proto"
    Syntax  string     // syntax 声明，如 "proto3"
    Defs    []TopDef   // 所有顶层定义
}

type TopDef struct {
    Kind string    // "message" 或 "enum"
    Name string    // 定义名称，如 "PlayerInfo"
    Text string    // 原始文本块（含花括号）
    Refs []string  // 引用的类型名列表，如 ["ItemInfo", "google.protobuf.Timestamp"]
}
```

### 解析流程

1. 读取文件内容
2. `StripComments` → 去注释后提取 `syntax` 和 `package`
3. `ScanTopLevelBlocks` → 在原始内容中扫描顶层 message/enum 块，返回 `[]Block`（位置信息）
4. 对每个 Block，提取块体 → `ExtractTypeRefs` 收集类型引用
5. 组装为 `PFile` 返回

### 示例

```protobuf
syntax = "proto3";
package game;

message PlayerInfo {
    string name = 1;
    int32 level = 2;
    repeated ItemInfo items = 3;
}

enum PlayerStatus {
    UNKNOWN = 0;
    ONLINE = 1;
}
```

输出：
```
PFile{
    Path:    "proto_src/player.proto",
    Package: "game",
    Syntax:  "proto3",
    Defs: [
        {Kind:"message", Name:"PlayerInfo", Text:"message PlayerInfo {...}", Refs:["ItemInfo"]},
        {Kind:"enum",    Name:"PlayerStatus", Text:"enum PlayerStatus {...}", Refs:[]},
    ],
}
```

---

## 环节 3.5：resolver（类型索引）— 构建类型查找表

**模块**: `converter/internal/resolver`（`TypeResolver`）

> 这一步在 pruner 的 `parseAndIndex` 内部调用，紧接在 parser 之后。

### 输入

| 参数 | 类型 | 说明 |
|------|------|------|
| `parsed` | `map[string]*model.PFile` | 所有已解析文件的映射（路径 → PFile） |

### 输出

| 返回值 | 类型 | 说明 |
|--------|------|------|
| `fullIndex` | `map[string]model.DefRef` | 全限定名索引：`"game.PlayerInfo"` → `{File, Def}` |
| `simpleIndex` | `map[string][]model.DefRef` | 简单名索引：`"PlayerInfo"` → `[{File, Def}, ...]` |

```go
type DefRef struct {
    File string    // 定义所在的文件路径
    Def  *TopDef   // 指向具体的 TopDef
}
```

### 示例

```
fullIndex: {
    "game.PlayerInfo":   {File:"player.proto", Def:&TopDef{Name:"PlayerInfo",...}},
    "game.PlayerStatus": {File:"player.proto", Def:&TopDef{Name:"PlayerStatus",...}},
    "game.ItemInfo":     {File:"item.proto",   Def:&TopDef{Name:"ItemInfo",...}},
}
simpleIndex: {
    "PlayerInfo":   [{File:"player.proto", ...}],
    "PlayerStatus": [{File:"player.proto", ...}],
    "ItemInfo":     [{File:"item.proto", ...}],
}
```

后续 `Resolve(curFile, curPkg, token)` 方法用这两个索引将类型名解析到具体定义。

---

## 环节 4：pruner — 裁剪定义

**模块**: `converter/internal/pruner`（`Pruner`）

### 总输入

| 参数 | 类型 | 来源 |
|------|------|------|
| `all` | `[]model.ProtoItem` | resolver.DepResolver 的输出 |
| `seeds` | `[]model.ProtoItem` | resolvedSeeds 或 normalized |
| `seedKeep` | `map[string]map[string]struct{}` | config.BuildSeedKeep 的输出 |
| `typeFieldKeep` | `map[string]map[string]struct{}` | config.BuildSeedKeep 的输出 |
| `opts` | `model.PruneOptions` | 裁剪配置（目录、语言、命名风格等） |

### 总输出

| 返回值 | 类型 | 说明 |
|--------|------|------|
| `outDir` | `string` | 输出目录路径 |
| `targets` | `[]model.ProtoItem` | 实际写出的目标文件列表 |

### 内部三阶段

#### 阶段 1：parseAndIndex

```
输入: all []ProtoItem
输出: parseResult{
    parsed:      map[path]*PFile,       // 所有文件的解析结果
    fullIndex:   map[string]DefRef,     // 全限定名索引
    simpleIndex: map[string][]DefRef,   // 简单名索引
}
```

#### 阶段 2：collectSelectedDefs

```
输入: parseResult + seeds + seedKeep + typeFieldKeep + inDir
输出: selected map[string]map[string]struct{}
       // 文件路径 → 该文件中需要保留的定义名集合
```

算法：
1. 从种子文件出发，根据 seedKeep 选择初始定义
2. BFS 遍历：对每个选中的定义，收集其类型引用 → 通过 Resolve 找到被引用的定义 → 加入队列
3. 过程中按 typeFieldKeep 裁剪 message 字段（减少不必要的类型引用传播）

示例：
```
seedKeep 指定保留 player.proto 的 PlayerInfo
→ PlayerInfo 引用了 ItemInfo
→ ItemInfo 被自动加入 selected
→ 最终 selected = {
    "player.proto": {"PlayerInfo"},
    "item.proto":   {"ItemInfo"},
  }
```

#### 阶段 3：assembleAndWrite

```
输入: parseResult + selected + typeFieldKeep + opts
输出: outDir string, targets []ProtoItem
      + 磁盘上的 .proto 文件
```

对每个文件：
1. 无选中定义 → 写 stub 文件（仅 syntax + package + namespace option）
2. 有选中定义 → 执行以下处理链：

```
原始定义文本
  │
  ├── PruneMessageFields(def, keepSet)    ← 按 typeFieldKeep 裁剪字段
  ├── StripSelfPackageQualifiers(def, pkg) ← 移除自包前缀（game.Foo → Foo）
  ├── TransformFieldNames(def, caseKind)   ← 字段名风格转换
  │
  ▼
裁剪后的定义文本
  │
  ├── 计算 crossImports（跨文件 import）
  ├── 计算 googleImports（Well-Known Types import）
  │
  ▼
组装完整文件：syntax + package + imports + namespace option + 定义
  │
  ├── Sanitize()  ← 移除注释、reserved 行、规范化空行
  │
  ▼
写入磁盘
```

---

## 环节 5：formatter — 格式化输出

**模块**: `converter/internal/formatter`（`OutputFormatter`）

> formatter 的方法在 pruner 的 assembleAndWrite 阶段被调用，不是独立的流水线环节，而是嵌入在 pruner 内部。

### Sanitize

| 输入 | 类型 | 说明 |
|------|------|------|
| `src` | `string` | 组装后的完整 proto 文件文本 |

| 输出 | 类型 | 说明 |
|------|------|------|
| 返回值 | `string` | 清理后的文本 |

处理链：
```
原始文本
  → stripCommentsOut()          移除注释
  → dropReservedLines()         移除 reserved 行
  → normalizeBlankLines()       合并连续空行
  → dropBlankLinesInsideTopBlocks()  移除块内空行
  → tightenBlockBlankLines()    修复花括号附近空行
  → normalizeBlankLines()       最终规范化
```

### StripSelfPackageQualifiers

| 输入 | 输出 | 示例 |
|------|------|------|
| `content="game.PlayerInfo field"`, `selfPkg="game"` | `"PlayerInfo field"` | 移除当前包的冗余限定 |

### TransformFieldNames

| 输入 | 输出 | 示例 |
|------|------|------|
| `def="message M { string user_name = 1; }"`, `caseKind="camel"` | `"message M { string UserName = 1; }"` | 按风格转换字段名 |

### WriteLangNamespaceOption

| 输入 | 输出 |
|------|------|
| `lang="csharp"`, `ns="Game.Proto"` | 写入 `option csharp_namespace = "Game.Proto";` |
| `lang="go"`, `ns="game/proto"` | 写入 `option go_package = "game/proto";` |
| `lang="lua"` | 不写任何 option |

---

## 完整数据流示例

假设配置要求从 `player.proto` 中只保留 `PlayerInfo` 的 `name` 和 `level` 字段：

```
1. config.Load("config.yaml")
   → Config{Import:{Dir:"proto_src", Keep:{Files:[{File:"player.proto", Keep:["PlayerInfo"]}], Types:[{Type:"PlayerInfo", Keep:["name","level"]}]}}}

2. config.BuildSeedKeep(cfg)
   → seeds: [{Path:"player.proto"}]
   → seedKeep: {"player.proto": {"PlayerInfo":{}}}
   → typeFieldKeep: {"PlayerInfo": {"name":{}, "level":{}}}

3. resolver.CollectWithImportsAndRoots(seeds, "proto_src")
   → all: [common.proto, item.proto, player.proto]  （player.proto import 了 item.proto 和 common.proto）
   → resolvedSeeds: [{Path:"proto_src/player.proto"}]

4. pruner.parseAndIndex(all)
   → parsed: {3 个文件的 PFile}
   → fullIndex/simpleIndex: {所有类型的索引}

5. pruner.collectSelectedDefs(...)
   → PlayerInfo 引用了 ItemInfo → ItemInfo 也被选中
   → selected: {"player.proto":{"PlayerInfo"}, "item.proto":{"ItemInfo"}}

6. pruner.assembleAndWrite(...)
   → PlayerInfo 被裁剪为只保留 name 和 level 字段
   → 自动生成 import "item.proto"
   → Sanitize 清理格式
   → 写入 output/player.proto 和 output/item.proto
```
