# util 子包

**包路径**: `converter/core/util`

**职责**: 提供跨包共享的工具函数，消除功能层包之间的重复代码。作为 `core/` 基础层的一部分，仅依赖标准库，不包含任何业务逻辑。

## 文件列表

### util.go — 共享工具函数

| 函数 | 说明 |
|------|------|
| `BaseName(tok string) string` | 从可能带包名限定的类型标记中提取简单名称 |

**BaseName 算法**:
1. 去除前导空白和前导 `.`
2. 空字符串直接返回
3. 查找最后一个 `.` 的位置，返回其后的部分
4. 无 `.` 时返回原字符串

**示例**:
- `"google.protobuf.Timestamp"` → `"Timestamp"`
- `".pkg.MyMessage"` → `"MyMessage"`
- `"MyMessage"` → `"MyMessage"`
- `""` → `""`

## 依赖

仅依赖标准库 `strings`，无任何内部包依赖。

## 设计说明

- `BaseName` 的规范实现位于此包，`resolver` 包中保留了一个同名的委托函数（标记为 `Deprecated`）用于向后兼容
- `resolver` 和 `pruner` 包均通过 `util.BaseName` 调用此函数，消除了之前两个包之间的重复实现
- 该包是 `core/model` 之外的另一个叶子包，位于依赖图的最底层
