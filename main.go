package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aura-studio/proto-converter/converter"
)

func main() {
	configPath := flag.String("c", filepath.FromSlash("template.proto.yaml"), "YAML 配置文件路径（相对运行目录）")
	flag.StringVar(configPath, "config", *configPath, "YAML 配置文件路径（同 -c）")
	workdir := flag.String("w", ".", "工作目录（相对运行目录），YAML 中的路径以此为基准")
	flag.StringVar(workdir, "workdir", *workdir, "工作目录（同 -w）")
	flag.Parse()

	startWD, _ := os.Getwd()
	configAbs := *configPath
	if !filepath.IsAbs(configAbs) {
		configAbs = filepath.Clean(filepath.Join(startWD, configAbs))
	}
	workAbs := *workdir
	if !filepath.IsAbs(workAbs) {
		workAbs = filepath.Clean(filepath.Join(startWD, workAbs))
	}
	if err := os.Chdir(workAbs); err != nil {
		fmt.Fprintf(os.Stderr, "无法进入工作目录: %s (%v)\n", workAbs, err)
		return
	}

	exp := &converter.Exporter{}

	exp.ConfigPath = configAbs
	if err := exp.Run(); err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}
}
