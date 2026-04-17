# proto-converter 项目文档

## 项目简介

`proto-converter` 是一个用 Go 编写的 Protocol Buffers 文件裁剪与转换工具。它从 YAML 配置文件读取规则，扫描源 `.proto` 文件，根据种子文件和保留规则裁剪出所需的定义及其依赖，最终输出经过命名风格转换和格式清理的 `.proto` 文件。

主要功能：
- 按种子文件和保留规则裁剪 message/enum 定义
- 自动追踪类型引用的传递依赖
- 支持文件名和字段名的命名风格转换（camel、snake、compact、keep）
- 支持多语言命名空间 option 写入（C#、Go、Lua）
- 支持 dry-run 模式预览操作
- 自动处理 Google Well-Known Types 的 import

## 目录结构

```
proto-converter/
├── main.go                     # 程序入口
├── go.mod                      # Go 模块定义
├── template.proto.yaml         # YAML 配置模板
├── scripts/
│   └── build.sh                # 多平台构建脚本
├── doc/                        # 项目文档
│   ├── README.md               # 本文件 - 项目总览
│   ├── architecture.md         # 架构设计说明
│   ├── dependency-graph.md     # 模块依赖图（Mermaid）
│   ├── model.md                # core/model 子包文档
│   ├── config.md               # internal/config 子包文档
│   ├── parser.md               # internal/parser 子包文档
│   ├── resolver.md             # internal/resolver 子包文档
│   ├── pruner.md               # internal/pruner 子包文档
│   ├── formatter.md            # internal/formatter 子包文档
│   ├── util.md                 # core/util 子包文档
│   ├── contract.md             # core/contract 子包文档
│   └── exporter.md             # exporter 编排层文档
└── converter/                  # 核心代码（三级嵌套结构）
    ├── core/                   # 基础层 — 零业务依赖的共享基础设施
    │   ├── model/              # 共享数据类型（叶子包，无外部依赖）
    │   │   ├── model.go
    │   │   └── case.go
    │   ├── util/               # 共享工具函数（仅依赖标准库）
    │   │   └── util.go
    │   └── contract/           # 核心接口定义（仅依赖 core/model + 标准库）
    │       └── contract.go
    ├── internal/               # 功能层 — 内部实现模块（Go internal 可见性限制）
    │   ├── config/             # 配置加载与校验（使用 Viper）
    │   │   ├── loader.go
    │   │   ├── types.go
    │   │   └── validator.go
    │   ├── parser/             # .proto 文件解析
    │   │   ├── parser.go
    │   │   ├── scanner.go
    │   │   └── types.go
    │   ├── resolver/           # 类型引用与 import 依赖解析
    │   │   ├── type.go
    │   │   ├── dep.go
    │   │   └── wellknown.go
    │   ├── pruner/             # 定义裁剪与编排
    │   │   ├── pruner.go
    │   │   ├── defpruner.go
    │   │   └── helpers.go
    │   └── formatter/          # 输出格式化与清理
    │       ├── sanitizer.go
    │       ├── writer.go
    │       └── transform.go
    └── exporter/               # 编排层 — 顶层入口，组装所有模块
        └── exporter.go
```

## 使用方式

```bash
# 基本用法
./proto-converter -c config.yaml -w /path/to/workdir

# 参数说明
#   -c, --config   YAML 配置文件路径（相对运行目录），默认 template.proto.yaml
#   -w, --workdir  工作目录，YAML 中的相对路径以此为基准，默认当前目录
```

## 构建

```bash
bash scripts/build.sh
```

会在 `build/` 目录下生成 5 个平台的可执行文件：
- `proto-converter-darwin-amd64` (macOS Intel)
- `proto-converter-darwin-arm64` (macOS Apple Silicon)
- `proto-converter-linux-amd64`
- `proto-converter-linux-arm64`
- `proto-converter-windows-amd64.exe`

## 文档索引

| 文档 | 说明 |
|------|------|
| [dependency-graph.md](dependency-graph.md) | 模块依赖图（Mermaid） |
| [architecture.md](architecture.md) | 整体架构、数据流、依赖关系 |
| [model.md](model.md) | core/model 子包：共享数据类型和工具函数 |
| [util.md](util.md) | core/util 子包：共享工具函数 |
| [contract.md](contract.md) | core/contract 子包：核心接口定义 |
| [config.md](config.md) | internal/config 子包：配置加载（Viper）、校验、种子构建 |
| [parser.md](parser.md) | internal/parser 子包：.proto 文件解析 |
| [resolver.md](resolver.md) | internal/resolver 子包：类型引用解析和 import 依赖解析 |
| [pruner.md](pruner.md) | internal/pruner 子包：定义裁剪编排 |
| [formatter.md](formatter.md) | internal/formatter 子包：输出格式化与清理 |
| [exporter.md](exporter.md) | exporter 编排层：导出流程编排 |
