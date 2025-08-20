package converter

// Cleaner 负责导出目录的后处理（重命名与清理）。
type Cleaner struct {
	OutDir string
	DryRun bool
}

// RenameToCamelCase 将 .cs 文件统一改为驼峰名。
func (c Cleaner) RenameToCamelCase() error {
	return renameToCamelCase(c.OutDir, c.DryRun)
}

// DeleteExtras 删除导出目录中不在期望列表内的文件。
func (c Cleaner) DeleteExtras(expected map[string]struct{}) error {
	return deleteExtras(c.OutDir, expected, c.DryRun)
}
