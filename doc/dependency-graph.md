# 模块依赖图

## 包级依赖关系

```mermaid
graph TD
    main["main.go"]
    converter["converter<br/><small>exporter · interfaces · types · case</small>"]
    config["converter/config<br/><small>loader · validator · types</small>"]
    parser["converter/parser<br/><small>parser · scanner · types</small>"]
    resolver["converter/resolver<br/><small>type · dep · wellknown</small>"]
    pruner["converter/pruner<br/><small>pruner · defpruner · helpers</small>"]
    formatter["converter/formatter<br/><small>sanitizer · writer · transform</small>"]
    model["converter/model<br/><small>model · case</small>"]

    main --> converter
    converter --> config
    converter --> parser
    converter --> resolver
    converter --> pruner
    converter --> formatter
    converter --> model
    config -.- model
    parser --> model
    resolver --> model
    pruner --> model
    pruner --> resolver
    formatter --> model

    style model fill:#e8f5e9,stroke:#388e3c
    style converter fill:#e3f2fd,stroke:#1565c0
    style main fill:#fff3e0,stroke:#ef6c00
    style config fill:#fce4ec,stroke:#c62828
```

> 虚线 `config -.- model` 表示 config 包**不**依赖 model（它内联了自己的 `SeedItem` 类型和 `normalizeSeedList` 函数以避免循环依赖）。

## 依赖矩阵

| 包 ↓ 依赖 → | model | config | parser | resolver | pruner | formatter |
|:---|:---:|:---:|:---:|:---:|:---:|:---:|
| **main** | | | | | | |
| **converter** | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **config** | | | | | | |
| **parser** | ✓ | | | | | |
| **resolver** | ✓ | | | | | |
| **pruner** | ✓ | | | ✓ | | |
| **formatter** | ✓ | | | | | |

## 设计要点

- **model** 是唯一的叶子包，不依赖任何其他内部包，所有共享类型都定义在这里
- **config** 是独立的，不依赖 model 也不依赖其他子包（通过内联类型避免循环依赖）
- **pruner → resolver** 是子包之间唯一的直接依赖，用于访问 `WellKnownTypes` 常量表
- **converter** 是唯一的组装点，汇聚所有子包
- **main** 只依赖 converter 顶层包
