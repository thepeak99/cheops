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
	Clone(commit *CommitInfo, targetDir string) error
	RegisterRepo(repo *config.Repository) error
}

type Cheops interface {
	Config() *config.CheopsConfig
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
	Containers []*Container
	Actions    []*Action
	Notifiers  []*Notifier
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
