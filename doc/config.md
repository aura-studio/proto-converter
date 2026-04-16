# config 子包

**包路径**: `converter/config`

**职责**: 负责 YAML 配置文件的读取、反序列化、校验和种子数据构建。将原来 `readProtoConfig` 一个函数承担的三重职责拆分为独立的 Loader 和 Validator。

## 文件列表

### types.go — 配置结构体定义

定义与 YAML 配置文件对应的 Go 结构体，所有字段都带有 `yaml` tag：

| 类型 | 说明 |
|------|------|
| `Config` | 顶层配置，包含 `DryRun`、`Import`、`Export` |
| `ImportSection` | 导入配置：`Dir`（源码根目录）、`Prune`（是否裁剪）、`Keep`（保留规则） |
| `ExportSection` | 导出配置：`Dir`（输出目录）、`Language`（目标语言）、`Namespace`（命名空间）、`FileNameCase`、`FieldNameCase` |
| `ImportKeep` | 保留规则：`Files`（文件级）和 `Types`（类型级） |
| `FileRule` | 文件级规则：`File`（proto 文件名）和 `Keep`（保留的定义名列表） |
| `TypeRule` | 类型级规则：`Type`（message 类型名）和 `Keep`（保留的字段名列表） |

### loader.go — 配置文件加载

**`Loader` 结构体**（空结构体）:

| 方法 | 说明 |
|------|------|
| `Load(path string) (Config, error)` | 读取 YAML 文件并反序列化为 Config。文件不存在或格式错误时返回描述性错误 |

仅负责 I/O 和反序列化，不做任何业务校验。

### validator.go — 配置校验与种子构建

**`Validator` 结构体**（空结构体）:

| 方法 | 说明 |
|------|------|
| `Validate(cfg Config) error` | 校验配置完整性。检查是否为空配置（所有关键字段均为零值时报错） |
| `BuildSeedKeep(cfg Config) ([]SeedItem, seedKeep, typeFieldKeep, error)` | 从配置构建种子列表和保留规则映射 |

**`SeedItem` 结构体**: config 包内部的种子文件表示，与 `model.ProtoItem` 结构相同但独立定义，避免 config 包依赖 model 以外的包。

**`normalizeSeedList` 内部函数**: 规范化种子文件名（补 `.proto` 后缀、去路径前缀、按 basename 去重）。

**BuildSeedKeep 返回值说明**:
- `[]SeedItem`: 规范化后的种子文件列表
- `seedKeep`: `map[文件路径]map[定义名]struct{}`，文件级保留规则
- `typeFieldKeep`: `map[类型名]map[字段名]struct{}`，类型级字段保留规则
