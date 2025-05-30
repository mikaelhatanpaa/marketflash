package config

// Config: PORT, ENVIRONMENT; DEBUG
// 1) Defaults 2) yaml file 3) ENV
// Validate

type Config struct {
	Port        int    `yaml:"port"`
	Environment string `yaml:"environment"`
	Debug       bool   `yaml:"debug"`
}
