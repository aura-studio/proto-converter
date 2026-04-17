package config

// FileRule describes one seed proto file and its kept fields.
type FileRule struct {
	File string   `mapstructure:"file"`
	Keep []string `mapstructure:"keep"`
}

// TypeRule describes fields to keep for a specific message type.
type TypeRule struct {
	Type string   `mapstructure:"type"`
	Keep []string `mapstructure:"keep"`
}

// ImportKeep contains file-level and type-level keep rules.
type ImportKeep struct {
	Files []FileRule `mapstructure:"files"`
	Types []TypeRule `mapstructure:"types"`
}

// ImportSection describes the import configuration.
type ImportSection struct {
	Dir   string     `mapstructure:"dir"`
	Prune *bool      `mapstructure:"prune"`
	Keep  ImportKeep `mapstructure:"keep"`
}

// ExportSection describes the export configuration.
type ExportSection struct {
	Dir           string `mapstructure:"dir"`
	Language      string `mapstructure:"language"`
	Namespace     string `mapstructure:"namespace"`
	FileNameCase  string `mapstructure:"fileNameCase"`
	FieldNameCase string `mapstructure:"fieldNameCase"`
}

// Config represents the complete YAML configuration.
type Config struct {
	DryRun *bool         `mapstructure:"dryRun"`
	Import ImportSection `mapstructure:"import"`
	Export ExportSection `mapstructure:"export"`
}
