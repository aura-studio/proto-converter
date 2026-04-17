# formatter 子包

**包路径**: `converter/internal/formatter`

**职责**: 负责裁剪后 proto 文件的输出格式化和清理，包括注释移除、reserved 行移除、空行规范化、命名空间 option 写入、字段名转换和自包前缀移除。

> 注意：该包位于 `internal/` 目录下，受 Go 语言 `internal` 可见性限制，仅允许 `converter/` 内部的包导入。实现 `core/contract.Formatter` 接口。

## 文件列表

### sanitizer.go — 输出清理主流程

**本地接口**: `ScannerIface` — 仅包含 `ScanTopLevelBlocks` 方法，用于扫描顶层块（在 `dropBlankLinesInsideTopBlocks` 中使用）。

**`OutputFormatter` 结构体**:

| 字段 | 类型 | 说明 |
|------|------|------|
| `Parser` | `ScannerIface` | 用于扫描顶层块（在 `dropBlankLinesInsideTopBlocks` 中使用） |

| 方法 | 说明 |
|------|------|
| `Sanitize(src string) string` | 对 proto 输出执行完整的清理流程 |

**Sanitize 处理流程**:
1. `stripCommentsOut` — 移除注释（保留字符串字面量中的内容）
2. `dropReservedLines` — 移除 `reserved` 行
3. `normalizeBlankLines` — 合并连续空行为单个空行
4. `dropBlankLinesInsideTopBlocks` — 移除 message/enum 块内部的空行
5. `tightenBlockBlankLines` — 修复花括号附近的多余空行
6. `normalizeBlankLines` — 最终再规范化一次

**内部函数**:

| 函数 | 说明 |
|------|------|
| `stripCommentsOut(s) string` | 状态机实现的注释移除，正确处理字符串字面量中的 `//` 和 `/*` |
| `dropReservedLines(s) string` | 用正则 `reservedLineRe` 匹配并移除 `reserved ...;` 行 |
| `normalizeBlankLines(s) string` | 合并连续空行，去除尾部空行，确保以换行结尾 |
| `tightenBlockBlankLines(s) string` | 移除 `{` 后和 `}` 前的多余空行 |
| `dropBlankLinesInsideTopBlocks(s) string` | 扫描顶层块，移除块体内的空行（从后往前替换避免索引位移） |

**依赖**: `converter/core/model`、标准库。

### writer.go — 命名空间 option 写入

| 方法 | 说明 |
|------|------|
| `WriteLangNamespaceOption(b *strings.Builder, lang, ns string)` | 根据目标语言写入对应的 option 语句 |

**语言映射**:
- `csharp`/`cs`/`c#` → `option csharp_namespace = "...";`
- `golang`/`go` → `option go_package = "...";`
- `lua` → 不写任何 option
- 其他 → 默认写 `csharp_namespace`

### transform.go — 字段名转换与自包前缀移除

| 方法 | 说明 |
|------|------|
| `StripSelfPackageQualifiers(content, selfPkg string) string` | 移除当前包的冗余限定前缀。例如在 `package foo` 的文件中，`foo.Bar` → `Bar` |
| `TransformFieldNames(def, caseKind string) string` | 按指定命名风格转换 message 字段名 |

**TransformFieldNames 算法**:
- `keep` 风格直接返回原文
- 定位 message 的 `{...}` 范围
- 用正则匹配字段声明行，提取字段名部分
- 调用 `model.ToCase` 转换字段名
- 保持类型名和字段编号不变
