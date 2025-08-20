package converter

import (
	bufio "bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// 通用辅助方法。

// protoItem 表示一个 proto 文件项的归一化信息。
type protoItem struct {
	Path string
	Dir  string
	Base string
}

// ensureDir 创建目录（若 dry 则仅打印）。
func ensureDir(dir string, dry bool) error {
	if dry {
		fmt.Printf("[dry] mkdir -p %s\n", dir)
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

// deleteExtras 删除导出目录中非期望的文件。
func deleteExtras(dir string, allowed map[string]struct{}, dry bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.EqualFold(filepath.Ext(name), ".cs") {
			continue
		}
		if _, ok := allowed[name]; ok {
			continue
		}
		p := filepath.Join(dir, name)
		if dry {
			fmt.Printf("[dry] rm extra %s\n", p)
			continue
		}
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// renameToCamelCase 将导出目录中的 .cs 文件重命名为驼峰形式。
func renameToCamelCase(dir string, dry bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.EqualFold(filepath.Ext(name), ".cs") {
			continue
		}
		camel := snakeToCamel(trimExt(name)) + ".cs"
		if name == camel {
			continue
		}
		oldPath := filepath.Join(dir, name)
		newPath := filepath.Join(dir, camel)
		if dry {
			fmt.Printf("[dry] mv %s %s\n", oldPath, newPath)
			continue
		}
		if strings.EqualFold(name, camel) && name != camel {
			tmp := filepath.Join(dir, ".tmp_"+camel)
			if err := os.Rename(oldPath, tmp); err != nil {
				return err
			}
			if err := os.Rename(tmp, newPath); err != nil {
				return err
			}
			continue
		}
		if err := os.Rename(oldPath, newPath); err != nil {
			return err
		}
	}
	return nil
}

// snakeToCamel 将蛇形/短横/点分名称转为驼峰形式。
func snakeToCamel(s string) string {
	s = strings.TrimSpace(s)
	b := strings.Builder{}
	upperNext := true
	for _, r := range s {
		if r == '_' || r == '-' || r == '.' || r == ' ' {
			upperNext = true
			continue
		}
		if upperNext {
			if 'a' <= r && r <= 'z' {
				r = r - ('a' - 'A')
			}
			upperNext = false
		}
		b.WriteRune(r)
	}
	return b.String()
}

// buildArgs 生成直接模式的 protogen 调用参数。
func buildArgs(outDir string, target protoItem, all []protoItem) []string {
	protoPaths := map[string]struct{}{}
	add := func(p string) {
		if p != "" {
			protoPaths[filepath.ToSlash(p)] = struct{}{}
		}
	}
	add(".")
	add(target.Dir)
	for _, it := range all {
		add(it.Dir)
	}
	if exists(filepath.FromSlash("external/proto/shared")) {
		add(filepath.FromSlash("external/proto/shared"))
	}
	if exists(filepath.FromSlash("external/proto/cli")) {
		add(filepath.FromSlash("external/proto/cli"))
	}
	args := make([]string, 0, 4+len(protoPaths))
	for p := range protoPaths {
		args = append(args, "--proto_path="+p)
	}
	args = append(args, "--csharp_out="+filepath.ToSlash(outDir))
	args = append(args, target.Base)
	return args
}

// buildArgsForTemp 生成临时目录模式的 protogen 调用参数。
func buildArgsForTemp(outDir, tempRoot string, target protoItem) []string {
	return []string{"--proto_path=" + filepath.ToSlash(tempRoot), "--csharp_out=" + filepath.ToSlash(outDir), target.Base}
}

// readList 读取旧版 TXT 配置。
func readList(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("无法打开配置 %s: %w", path, err)
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	var list []string
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.IndexByte(line, '#'); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line != "" {
			list = append(list, line)
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, errors.New("配置为空: 未指定任何 proto")
	}
	return list, nil
}

// normalizeItem 将传入条目规整为 protoItem。
func normalizeItem(s string) (protoItem, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "./")
	s = strings.TrimPrefix(s, "/")
	s = strings.TrimPrefix(s, "\\")
	if s == "" {
		return protoItem{}, errors.New("空的 proto 条目")
	}
	base := filepath.Base(s)
	dir := filepath.Dir(s)
	if dir == "." {
		dir = ""
	}
	return protoItem{Path: s, Dir: dir, Base: base}, nil
}

// findRepoRoot 尝试定位仓库根目录。
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

func exists(p string) bool { _, err := os.Stat(p); return err == nil }
func cd(dir string)        { _ = os.Chdir(dir) }

// shortPath 将路径转换为相对仓库的短路径（Windows 使用正斜杠）。
func shortPath(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	root := "."
	if r, err := findRepoRoot(); err == nil {
		root = r
	}
	if rel, err := filepath.Rel(root, p); err == nil {
		p = rel
	}
	if runtime.GOOS == "windows" {
		p = filepath.ToSlash(p)
	}
	return p
}

func trimExt(name string) string { return strings.TrimSuffix(name, filepath.Ext(name)) }

// process execution
// run 以子进程执行命令，继承标准输出/错误。
func run(cmd string, args ...string) error {
	fmt.Printf("  $ %s %s\n", shortPath(cmd), strings.Join(args, " "))
	c := execCmd(cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

type execCmdT struct {
	Command        string
	Args           []string
	Stdout, Stderr *os.File
}

func execCmd(name string, args ...string) *execCmdT { return &execCmdT{Command: name, Args: args} }

func (c *execCmdT) Run() error { return runCommand(c.Command, c.Args, c.Stdout, c.Stderr) }

// runCommand 执行外部命令并等待完成。
func runCommand(path string, args []string, stdout, stderr *os.File) error {
	argv0, err := filepath.Abs(path)
	if err != nil {
		argv0 = path
	}
	attr := &os.ProcAttr{Files: []*os.File{os.Stdin, stdout, stderr}}
	proc, err := os.StartProcess(argv0, append([]string{argv0}, args...), attr)
	if err != nil {
		return err
	}
	state, err := proc.Wait()
	if err != nil {
		return err
	}
	if !state.Success() {
		return fmt.Errorf("exit status: %v", state)
	}
	return nil
}

// check 为简单错误检查助手。
func check(err error) {
	if err != nil {
		fatalf("%v", err)
	}
}
func fatalf(format string, a ...any) { fmt.Fprintf(os.Stderr, format+"\n", a...); os.Exit(1) }

// regexes reused
var importRe = regexp.MustCompile(`(?m)^\s*import\s+\"([^\"]+)\"\s*;`)
