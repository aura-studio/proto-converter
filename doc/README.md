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
│   ├── model.md                # model 子包文档
│   ├── config.md               # config 子包文档
│   ├── parser.md               # parser 子包文档
│   ├── resolver.md             # resolver 子包文档
│   ├── pruner.md               # pruner 子包文档
│   ├── formatter.md            # formatter 子包文档
│   └── converter.md            # converter 顶层包文档
└── converter/                  # 核心代码
    ├── exporter.go             # 导出流程编排
    ├── interfaces.go           # 核心接口定义
    ├── types.go                # 类型别名（从 model 重导出）
    ├── case.go                 # 命名风格转换（顶层包副本）
    ├── case_test.go            # 命名风格单元测试
    ├── model/                  # 共享数据模型（无外部依赖的叶子包）
    ├── config/                 # YAML 配置加载与校验
    ├── parser/                 # .proto 文件解析
    ├── resolver/               # 类型引用与 import 依赖解析
    ├── pruner/                 # 定义裁剪与编排
    └── formatter/              # 输出格式化与清理
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
| [architecture.md](architecture.md) | 整体架构、数据流、依赖关系 |
| [converter.md](converter.md) | converter 顶层包：Exporter、接口、类型别名 |
| [model.md](model.md) | model 子包：共享数据类型和工具函数 |
| [config.md](config.md) | config 子包：YAML 配置加载、校验、种子构建 |
| [parser.md](parser.md) | parser 子包：.proto 文件解析 |
| [resolver.md](resolver.md) | resolver 子包：类型引用解析和 import 依赖解析 |
| [pruner.md](pruner.md) | pruner 子包：定义裁剪编排 |
| [formatter.md](formatter.md) | formatter 子包：输出格式化与清理 |
