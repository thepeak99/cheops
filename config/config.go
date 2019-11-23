package config

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type General struct {
	WebhookURL string `yaml:"webhook_url"`
	TLSCert    string `yaml:"tls_cert"`
	TLSKey     string `yaml:"tls_key"`
	BindAddr   string `yaml:"bind_addr"`
}

type Repository struct {
	Provider string
	URL      string
	Branch   string
	Secrets  map[string]interface{}
}

type GitProvider struct {
	Name     string
	Type     string
	Username string
	Password string
	SSHKey   string
	Token    string
}

type DockerCredsProvider struct {
	Name               string
	Type               string
	AwsRegion          string `yaml:"aws_region"`
	AwsAccessKeyID     string `yaml:"aws_access_key_id"`
	AwsSecretAccessKey string `yaml:"aws_secret_access_key"`
	AwsSessionToken    string `yaml:"aws_session_token"`
}

type Providers struct {
	Git         []*GitProvider
	DockerCreds []*DockerCredsProvider `yaml:"docker_creds"`
}

type CheopsConfig struct {
	General   General
	Repos     []*Repository
	Providers Providers
}

func parseConfig(data []byte) (*CheopsConfig, error) {
	config := CheopsConfig{}
	err := yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func LoadConfig() (*CheopsConfig, error) {
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
