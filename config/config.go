package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Old Clickhouse `yaml:"old"`
	New Clickhouse `yaml:"new"`
}

type Clickhouse struct {
	Address  string `yaml:"address"`
	Database string `yaml:"database"`
}

func New(filepath string) Config {
	var config Config

	bs, err := os.ReadFile(filepath)
	if err != nil {
		panic(fmt.Sprintf("failed to read config file, err: %v", err))
	}
	err = yaml.Unmarshal(bs, &config)
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal config file content, err: %v", err))
	}

	return config
}
