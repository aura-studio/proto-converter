package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/aura-studio/proto-converter/converter"
)

func main() {
	configPath := flag.String("c", filepath.FromSlash("template.proto.yaml"), "YAML 配置文件路径（相对运行目录）")
	flag.StringVar(configPath, "config", *configPath, "YAML 配置文件路径（同 -c）")
	workdir := flag.String("w", ".", "工作目录（相对运行目录），YAML 中的路径以此为基准")
	flag.StringVar(workdir, "workdir", *workdir, "工作目录（同 -w）")
	flag.Parse()

	cvt := &converter.Converter{
		ConfigPath: *configPath,
		WorkPath:   *workdir,
	}

	if err := cvt.Run(); err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}
}
