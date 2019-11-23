package config

import (
	"cheops/types"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

func parseConfig(data []byte) (*types.CheopsConfig, error) {
	config := types.CheopsConfig{}
	err := yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func LoadConfig() (*types.CheopsConfig, error) {
	var configBytes []byte
	var err error

	if configStr := os.Getenv("CHEOPS_CONFIG"); configStr != "" {
		configBytes = []byte(configStr)
	} else {
		configBytes, err = ioutil.ReadFile("cheops.yaml")
		if err != nil {
			return nil, err
		}
	}

	return parseConfig(configBytes)
}
