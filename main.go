package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aura-studio/proto-converter/converter"
)

func main() {
	// 仅保留配置文件参数，支持 -c 与 --config
	configPath := flag.String("c", filepath.FromSlash("external/proto/template.proto.yaml"), "YAML 配置文件路径（默认使用模板 template.proto.yaml）")
	// 兼容长名 --config
	flag.StringVar(configPath, "config", *configPath, "YAML 配置文件路径（同 -c）")
	flag.Parse()

	// 保持以仓库根为工作目录（与原有行为一致）
	if root, err := findRepoRoot(); err == nil {
		_ = os.Chdir(root)
	}

	// 构造并运行导出器（其余参数从配置文件中读取）
	exp := &converter.Exporter{
		// 其他参数从配置文件中读取
	}

	exp.ConfigPath = *configPath
	if err := exp.Run(); err != nil {
		// 标准错误输出并以非零退出
		fmt.Printf("错误: %v\n", err)
		return
	}

	fmt.Println("完成导出。")
}

// 其余组件（SeedLoader、DepResolver、Pruner 等）在同目录的独立文件中实现。

// findRepoRoot: 尝试自当前目录向上查找包含 external 与 go.mod 的目录
func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	cur := wd
	for i := 0; i < 6; i++ {
		if exists(filepath.Join(cur, "external")) && exists(filepath.Join(cur, "go.mod")) {
			return cur, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return wd, nil
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
