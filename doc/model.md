# model 子包

**包路径**: `converter/model`

**职责**: 定义所有子包共享的数据类型和工具函数。作为无外部依赖的叶子包，打破子包之间的循环依赖。

## 文件列表

### model.go — 共享数据类型与工具函数

**数据类型**:

| 类型 | 说明 |
|------|------|
| `ProtoItem` | 表示一个 proto 文件条目，包含 `Path`（完整路径）、`Dir`（目录）、`Base`（文件名） |
| `PFile` | 解析后的 proto 文件，包含 `Path`、`Package`、`Syntax` 和 `Defs`（顶层定义列表） |
| `TopDef` | 顶层定义（message 或 enum），包含 `Kind`、`Name`、`Text`（原始文本）、`Refs`（类型引用列表） |
| `DefRef` | 定义引用，指向某个文件中的某个 TopDef |
| `Block` | 扫描到的顶层块位置信息，包含 `Kind`、`Name`、`Start`、`BraceStart`、`End` |
| `PruneOptions` | 裁剪配置参数，封装 InDir、OutDir、Namespace、Language、FileNameCase、FieldNameCase、DryRun |

**工具函数**:

| 函数 | 说明 |
|------|------|
| `NormalizeItem(s string) (ProtoItem, error)` | 规范化 proto 文件路径字符串，去除前缀 `./`、`/`、`\`，拆分为 Path/Dir/Base |
| `Exists(p string) bool` | 判断文件或目录是否存在 |
| `ShortPath(p string) string` | 将路径转为正斜杠格式 |
| `TrimExt(name string) string` | 移除文件扩展名 |
| `EnsureDir(dir string, dry bool) error` | 创建目录（支持 dry-run 模式） |
| `BlockBody(src string, b Block) string` | 提取 Block 花括号内的文本 |
| `Block.FullText(src string) string` | 提取 Block 的完整源文本 |

### case.go — 命名风格转换

| 函数 | 说明 |
|------|------|
| `ToCase(s, caseKind string) string` | 按指定风格转换字符串。支持 `camel`（大驼峰）、`snake`（下划线）、`compact`（全小写无分隔）、`keep`（不变） |

内部辅助函数 `toCamel`、`toSnake`、`removeDelims` 等实现具体的转换逻辑。
