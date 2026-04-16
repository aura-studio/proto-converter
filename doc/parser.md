# parser 子包

**包路径**: `converter/parser`

**职责**: 将 `.proto` 文件解析为结构化的内部表示（`model.PFile`）。包含词法级的扫描器，能识别顶层 message/enum 定义块、剥离注释、提取类型引用。

## 文件列表

### types.go — 字符判断辅助函数

| 函数 | 说明 |
|------|------|
| `IsIdentStart(b byte) bool` | 判断字节是否可以作为标识符开头（字母或下划线） |
| `IsIdent(b byte) bool` | 判断字节是否可以出现在标识符中（字母、数字或下划线） |
| `IsSpace(b byte) bool` | 判断字节是否为空白字符（空格、制表符、回车、换行） |

这些函数被 scanner.go 中的扫描逻辑使用。

### parser.go — 文件解析入口

**`ProtoParser` 结构体**（空结构体，实现 `converter.Parser` 接口）:

| 方法 | 说明 |
|------|------|
| `ParseFile(path string) (*model.PFile, error)` | 读取并解析一个 .proto 文件 |

**ParseFile 处理流程**:
1. 读取文件内容
2. 调用 `StripComments` 去除注释，提取 `syntax` 和 `package` 声明
3. 调用 `ScanTopLevelBlocks` 扫描原始内容中的顶层 message/enum 块
4. 对每个块，提取块体并调用 `ExtractTypeRefs` 收集类型引用
5. 组装为 `model.PFile` 返回

### scanner.go — 扫描器

**`ProtoParser` 上的方法**:

| 方法 | 说明 |
|------|------|
| `ScanTopLevelBlocks(src string) []model.Block` | 扫描 proto 源码中所有顶层 message/enum 定义块 |
| `StripComments(src string) string` | 移除行注释（`//`）和块注释（`/* */`） |
| `ExtractTypeRefs(src string) []string` | 从去注释后的源码中提取字段的类型引用 |

**ScanTopLevelBlocks 算法**:
- 手写的单遍扫描器，跟踪花括号深度
- 仅在深度为 0 时识别 `message` 和 `enum` 关键字
- 正确处理字符串字面量、行注释、块注释中的花括号
- 嵌套的 message/enum 不会被识别为顶层块（因为深度 > 0）

**ExtractTypeRefs 算法**:
- 使用正则匹配字段声明模式 `[repeated|optional] TypeName fieldName = N`
- 对 `map<K, V>` 类型特殊处理，提取值类型 V
- 返回去重后的类型名列表
