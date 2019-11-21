package types

import (
	"cheops/config"
	"io"
)

// DockerCredsProvider provides credentials for pushing Docker images
type DockerCredsProvider interface {
	GetCredentials() (string, error)
}

// GitProvider provides cloning access to a repository
type GitProvider interface {
	Clone(repo *config.Repository, commit, targetDir string) error
	RegisterRepo(repo *config.Repository) error
}

type Cheops interface {
	Config() *config.CheopsConfig
	RegisterWebhook(endpoint string, webhook WebhookFunc)
	Execute(buildCtxt *BuildContext) error
	Serve() error
}

type BuildContext struct {
	Commit string
	Branch string
	Build  *config.Build
}

type WebhookFunc func(body io.ReadCloser, headers map[string][]string) (*BuildContext, error)
