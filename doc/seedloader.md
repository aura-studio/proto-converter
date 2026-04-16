# seedloader 子包

**包路径**: `converter/seedloader`

**职责**: 负责种子文件名的规范化和去重。将用户在配置中指定的文件名列表转换为标准化的 `ProtoItem` 列表。

## 文件列表

### seedloader.go

**`SeedLoader` 结构体**（空结构体，实现 `converter.SeedLoaderIface` 接口）:

| 方法 | 说明 |
|------|------|
| `SeedsFromList(list []string) ([]model.ProtoItem, error)` | 规范化种子文件名列表并去重 |

**SeedsFromList 处理流程**:
1. 遍历输入列表，跳过空字符串
2. 如果文件名不以 `.proto` 结尾，自动补充后缀
3. 调用 `model.NormalizeItem` 规范化路径
4. 调用 `dedupItems` 按 basename（不区分大小写）去重

**`dedupItems` 内部函数**: 按 `strings.ToLower(it.Base)` 去重，保留首次出现的条目。

> 注意：当前 config/validator.go 中的 `normalizeSeedList` 函数内联了相同的逻辑，不再直接调用此包。seedloader 包保留作为独立的可复用组件。
