package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Loader is responsible for reading and deserializing configuration files.
type Loader struct{}

// Load reads a configuration file using Viper and deserializes it into a Config.
func (Loader) Load(path string) (Config, error) {
	v := viper.New()
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	v.SetConfigName(name)
	v.SetConfigType(strings.TrimPrefix(ext, "."))
	v.AddConfigPath(dir)

	if err := v.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("无法打开配置 %s: %w", path, err)
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return Config{}, fmt.Errorf("配置解析失败: %w", err)
	}
	return c, nil
}
