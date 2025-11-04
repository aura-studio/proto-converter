package converter

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type Converter struct {
	ConfigPath string
	WorkPath   string
}

func (cvt Converter) Run() error {
	c := NewConfig(cvt.ConfigPath)
	fileKeeper := NewFileKeeper(c)
	typeKeeper := NewTypeKeeper(c)

	return nil
}

func (cvt Converter) preparePath() {
	wd, err := os.Getwd()
	if err != nil {
		log.Panicf("无法获取当前工作目录: %w", err)
	}
	if !filepath.IsAbs(cvt.ConfigPath) {
		cvt.ConfigPath = filepath.Clean(filepath.Join(wd, cvt.ConfigPath))
	}
	workPath := cvt.WorkPath
	if !filepath.IsAbs(workPath) {
		workPath = filepath.Clean(filepath.Join(wd, workPath))
	}
	cvt.WorkPath = workPath
	if err := os.Chdir(workPath); err != nil {
		fmt.Fprintf(os.Stderr, "无法进入工作目录: %s (%v)\n", workPath, err)
		return
	}
}
