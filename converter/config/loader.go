package config

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v3"
)

// Loader is responsible for reading and deserializing configuration files.
type Loader struct{}

// Load reads a YAML configuration file and deserializes it into a Config.
func (Loader) Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("无法打开配置 %s: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("配置解析失败: %w", err)
	}
	return c, nil
}
