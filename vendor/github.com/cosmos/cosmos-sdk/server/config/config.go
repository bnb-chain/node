package config

// BaseConfig defines the server's basic configuration
type BaseConfig struct {
}

// Config defines the server's top level configuration
type Config struct {
	BaseConfig `mapstructure:",squash"`
}

func DefaultConfig() *Config {
	return &Config{BaseConfig{}}
}

// Storage for init gen-tx command input parameters
type GenTx struct {
	Name      string
	CliRoot   string
	Overwrite bool
	IP        string
}
