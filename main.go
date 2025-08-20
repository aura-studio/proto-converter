package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aura-studio/proto-converter/converter"
)

func main() {
	// 参数：配置文件(-c/--config) 与 工作目录(-w/--workdir)
	configPath := flag.String("c", filepath.FromSlash("template.proto.yaml"), "YAML 配置文件路径（相对运行目录）")
	flag.StringVar(configPath, "config", *configPath, "YAML 配置文件路径（同 -c）")
	workdir := flag.String("w", ".", "工作目录（相对运行目录），YAML 中的路径以此为基准")
	flag.StringVar(workdir, "workdir", *workdir, "工作目录（同 -w）")
	flag.Parse()

	// 记录启动时的运行目录，用于正确解析相对传参
	startWD, _ := os.Getwd()
	// 将 config 解析为基于启动目录的绝对路径（不受后续 chdir 影响）
	configAbs := *configPath
	if !filepath.IsAbs(configAbs) {
		configAbs = filepath.Clean(filepath.Join(startWD, configAbs))
	}
	// 解析并切换到工作目录，使 YAML 内的相对路径以工作目录为基准
	workAbs := *workdir
	if !filepath.IsAbs(workAbs) {
		workAbs = filepath.Clean(filepath.Join(startWD, workAbs))
	}
	if err := os.Chdir(workAbs); err != nil {
		fmt.Fprintf(os.Stderr, "无法进入工作目录: %s (%v)\n", workAbs, err)
		return
	}

	// 构造并运行导出器（其余参数从配置文件中读取）
	exp := &converter.Exporter{}

	exp.ConfigPath = configAbs
	if err := exp.Run(); err != nil {
		// 标准错误输出并以非零退出
		fmt.Printf("错误: %v\n", err)
		return
	}

	fmt.Println("完成导出。")
}
