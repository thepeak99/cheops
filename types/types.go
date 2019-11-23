package types

import (
	"io"
)

type GeneralConfig struct {
	WebhookURL string `yaml:"webhook_url"`
	TLSCert    string `yaml:"tls_cert"`
	TLSKey     string `yaml:"tls_key"`
	BindAddr   string `yaml:"bind_addr"`
}

type GitProviderConfig struct {
	Name     string
	Type     string
	Username string
	Password string
	SSHKey   string
	Token    string
}

type DockerCredsProviderConfig struct {
	Name               string
	Type               string
	AwsRegion          string `yaml:"aws_region"`
	AwsAccessKeyID     string `yaml:"aws_access_key_id"`
	AwsSecretAccessKey string `yaml:"aws_secret_access_key"`
	AwsSessionToken    string `yaml:"aws_session_token"`
}

type ProvidersConfig struct {
	Git         []*GitProviderConfig
	DockerCreds []*DockerCredsProviderConfig `yaml:"docker_creds"`
}

type CheopsConfig struct {
	General   GeneralConfig
	Repos     []*Repository
	Providers ProvidersConfig
}

type Repository struct {
	Provider string
	URL      string
	Branch   string
	Secrets  map[string]interface{}
}

// DockerCredsProvider provides credentials for pushing Docker images
type DockerCredsProvider interface {
	GetCredentials() (string, error)
}

// GitProvider provides cloning access to a repository
type GitProvider interface {
	Clone(commit *CommitInfo, targetDir string) error
	RegisterRepo(repo *Repository) error
}

type Cheops interface {
	Config() *CheopsConfig
	RegisterWebhook(endpoint string, webhook WebhookFunc)
	Execute(buildCtxt *BuildContext) error
	Serve() error
}

type Container struct {
	Dockerfile string
	Context    string
	Tag        string
	Args       map[string]*string
}

type Action struct {
	Type      string
	Commands  []string
	Image     string
	Provider  string
	Container string
}

type Build struct {
	Name       string
	Branch     string
	Containers []*Container
	Actions    []*Action
	Notifiers  []*Notifier
}

type BuildsConfig struct {
	Builds []*Build
}

type Notifier struct{}

type CommitInfo struct {
	ID      string
	Branch  string
	RepoURL string
}

type BuildContext struct {
	Build   *Build
	Commit  *CommitInfo
	RepoDir string
}

type WebhookFunc func(body io.ReadCloser, headers map[string][]string) (*CommitInfo, error)
