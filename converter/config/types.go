package config

// FileRule describes one seed proto file and its kept fields.
type FileRule struct {
	File string   `yaml:"file"`
	Keep []string `yaml:"keep"`
}

// TypeRule describes fields to keep for a specific message type.
type TypeRule struct {
	Type string   `yaml:"type"`
	Keep []string `yaml:"keep"`
}

// ImportKeep contains file-level and type-level keep rules.
type ImportKeep struct {
	Files []FileRule `yaml:"files"`
	Types []TypeRule `yaml:"types"`
}

// ImportSection describes the import configuration.
type ImportSection struct {
	Dir   string     `yaml:"dir"`
	Prune *bool      `yaml:"prune"`
	Keep  ImportKeep `yaml:"keep"`
}

// ExportSection describes the export configuration.
type ExportSection struct {
	Dir           string `yaml:"dir"`
	Language      string `yaml:"language"`
	Namespace     string `yaml:"namespace"`
	FileNameCase  string `yaml:"fileNameCase"`
	FieldNameCase string `yaml:"fieldNameCase"`
}

// Config represents the complete YAML configuration.
type Config struct {
	DryRun *bool         `yaml:"dryRun"`
	Import ImportSection `yaml:"import"`
	Export ExportSection `yaml:"export"`
}
