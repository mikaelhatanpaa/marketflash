package config

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"

	"gopkg.in/yaml.v3"
)

var (
	ErrReadConfig   = errors.New("unable to read config file")
	ErrParseYAML    = errors.New("error parsing yaml")
	ErrInvalidPort  = errors.New("invalid port value")
	ErrInvalidDebug = errors.New("invalid debug value")

	ErrValidationFailed   = errors.New("config validation failed")
	ErrMissingDatabaseURL = errors.New("database_url is required")
	ErrInvalidPortRange   = errors.New("port must be between 1 and 65535")
	ErrMissingAPIKey      = errors.New("api key is missing")
	ErrInvalidEnvironment = errors.New("environment must be one of: development, staging, production")
)

var validEnvironments = []string{"development", "staging", "production"}

type config struct {
	DatabaseURL string `yaml:"database_url"`
	Port        int    `yaml:"port"`
	Environment string `yaml:"environment"`
	APIKey      string `yaml:"api_key"`
	Debug       bool   `yaml:"debug"`
}

func LoadConfig(cfgPath string) (config, error) {
	cfg := config{
		Port:        8080,
		Environment: "development",
		Debug:       false,
	}

	if cfgPath != "" {
		data, err := os.ReadFile(cfgPath) // data []byte

		if err != nil && !os.IsNotExist(err) {
			return config{}, fmt.Errorf("%w: %s", ErrReadConfig, err)
		}

		if err == nil && len(data) == 0 {
			return config{}, fmt.Errorf("%w: config file is empty", ErrReadConfig)
		}

		if err == nil && len(data) > 0 {
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return config{}, fmt.Errorf("%w: %s", ErrParseYAML, err)
			}
		}
	}

	if dbURL, ok := os.LookupEnv("DATABASE_URL"); ok {
		cfg.DatabaseURL = dbURL
	}

	if portStr, ok := os.LookupEnv("PORT"); ok {
		port, err := strconv.ParseInt(portStr, 10, 32)
		if err != nil || port < 1 || port > 65535 {
			return config{}, fmt.Errorf("%w: got %q", ErrInvalidPort, portStr)
		}
		cfg.Port = int(port)
	}

	if apiKey, ok := os.LookupEnv("API_KEY"); ok {
		cfg.APIKey = apiKey
	}

	if env, ok := os.LookupEnv("ENVIRONMENT"); ok {
		cfg.Environment = env
	}

	if debugStr, ok := os.LookupEnv("DEBUG"); ok {
		debug, err := strconv.ParseBool(debugStr)

		if err != nil {
			return config{}, fmt.Errorf("%w: got %q", ErrInvalidDebug, debugStr)
		}
		cfg.Debug = debug
	}

	if err := cfg.Validate(); err != nil {
		return config{}, fmt.Errorf("%w: %s", ErrValidationFailed, err.Error())
	}

	return cfg, nil
}

func (c config) Validate() error {
	var errs = make([]error, 0, 4)

	if c.DatabaseURL == "" {
		errs = append(errs, ErrMissingDatabaseURL)
	}

	if c.Port < 1 || c.Port > 65535 {
		errs = append(errs, fmt.Errorf("%w: got %d", ErrInvalidPortRange, c.Port))
	}

	if c.APIKey == "" {
		errs = append(errs, ErrMissingAPIKey)
	}

	if !slices.Contains(validEnvironments, c.Environment) {
		errs = append(errs, fmt.Errorf("%w: got %q", ErrInvalidEnvironment, c.Environment))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
