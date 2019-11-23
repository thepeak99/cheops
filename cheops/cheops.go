package cheops

import (
	"bytes"
	"cheops/aws"
	"cheops/config"
	"cheops/docker"
	"cheops/github"
	"cheops/types"
	"errors"
	"html/template"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type cheopsImpl struct {
	config               *config.CheopsConfig
	gitProviders         map[string]types.GitProvider
	dockerCredsProviders map[string]types.DockerCredsProvider
}

func (c *cheopsImpl) Config() *config.CheopsConfig {
	return c.config
}

func (c *cheopsImpl) initDockerCredsProvider(providerConfig *config.DockerCredsProvider) (types.DockerCredsProvider, error) {
	switch providerConfig.Type {
	case "aws":
		provider, err := aws.New(
			providerConfig.AwsRegion,
			providerConfig.AwsAccessKeyID,
			providerConfig.AwsSecretAccessKey,
			providerConfig.AwsSessionToken,
		)
		if err != nil {
			return nil, err
		}
		return provider, nil

	default:
		return nil, errors.New("Unsupported provider: " + providerConfig.Type)
	}
}

func (c *cheopsImpl) initGitProvider(providerConfig *config.GitProvider) (types.GitProvider, error) {
	var provider types.GitProvider
	var err error

	switch providerConfig.Type {
	case "github":
		provider, err = github.New(c, providerConfig)
		if err != nil {
			return nil, err
		}

	default:
		return nil, errors.New("Unsupported provider: " + providerConfig.Type)
	}

	return provider, nil
}

func New() types.Cheops {
	config, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	c := cheopsImpl{}
	c.config = config
	c.gitProviders = make(map[string]types.GitProvider)
	c.dockerCredsProviders = make(map[string]types.DockerCredsProvider)

	log.Debug("Initializing Git Providers")
	for _, gitProvider := range config.Providers.Git {
		c.gitProviders[gitProvider.Name], err = c.initGitProvider(gitProvider)
		if err != nil {
			log.WithFields(log.Fields{
				"provider": gitProvider.Name,
				"error":    err,
			}).Fatal("Error loading provider")
		}
	}

	log.Debug("Initializing Docker Credential providers")
	for _, dockerCredsProvider := range config.Providers.DockerCreds {
		c.dockerCredsProviders[dockerCredsProvider.Name], err = c.initDockerCredsProvider(dockerCredsProvider)
		if err != nil {
			log.WithFields(log.Fields{
				"provider": dockerCredsProvider.Name,
				"error":    err,
			}).Fatal("Error loading provider")
		}
	}

	log.Debug("Loading builds")
	for _, repo := range config.Repos {
		provider, ok := c.gitProviders[repo.Provider]
		if !ok {
			log.WithFields(log.Fields{
				"repo":     repo.URL,
				"provider": repo.Provider,
			}).Fatal("Unknown Git provider")
		}

		err := provider.RegisterRepo(repo)
		if err != nil {
			log.WithFields(log.Fields{
				"provider": repo.Provider,
				"repo":     repo.URL,
				"error":    err,
			}).Fatal("Can't register repository to provider")
		}
	}

	return &c
}

func (c *cheopsImpl) procAction(action *types.Action) error {
	log.WithFields(log.Fields{
		"type": action.Type,
	}).Debug("Performing action")

	switch action.Type {
	case "push":
		provider, ok := c.dockerCredsProviders[action.Provider]
		if !ok {
			return errors.New("Unknown provider: " + action.Provider)
		}

		log.WithFields(log.Fields{
			"provider": action.Provider,
		}).Debug("Getting Docker credentials")
		creds, err := provider.GetCredentials()
		if err != nil {
			return err
		}

		err = docker.PushImage(action.Image, creds)
		if err != nil {
			return err
		}

	case "exec":
		err := docker.RunContainer(action.Image, action.Commands, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func loadBuild(repoDir string, repo *config.Repository, commit *types.CommitInfo) (*types.Build, error) {
	tmpl, err := template.ParseFiles(repoDir + "/cheops.yaml")
	if err != nil {
		return nil, err
	}

	buf := bytes.Buffer{}
	data := map[string]interface{}{
		"Secrets":    repo.Secrets,
		"Commit":     commit.ID,
		"Repository": commit.RepoURL,
		"Branch":     commit.Branch,
	}
	tmpl.Execute(&buf, &data)
	
	configBytes := buf.Bytes()
	log.WithFields(log.Fields{
		"repo":   commit.RepoURL,
		"branch": commit.Branch,
		"commit": commit.ID,
	}).Debug(string(configBytes))

	config := types.Build{}
	err = yaml.Unmarshal(configBytes, &config)

	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *cheopsImpl) GetBuildContext(repo *config.Repository, commit *types.CommitInfo) (*types.BuildContext, error) {
	log.WithFields(log.Fields{
		"repo": repo.URL,
	}).Debug("Preparing build context")

	cloneDir, err := ioutil.TempDir("/tmp", "cheops")
	if err != nil {
		log.WithFields(log.Fields{
			"directory": cloneDir,
			"error":     err,
		}).Error("Can't create temporary directory")
		return nil, err
	}

	provider := c.gitProviders[repo.Provider]
	err = provider.Clone(commit, cloneDir)
	if err != nil {
		log.WithFields(log.Fields{
			"repository": repo.URL,
			"error":      err,
		}).Error("Can't clone repository")
		return nil, err
	}

	b, err := loadBuild(cloneDir, repo, commit)
	if err != nil {
		log.WithFields(log.Fields{
			"repository": repo.URL,
			"error":      err,
		}).Error("Can't load build")
		return nil, err
	}

	return &types.BuildContext{
		Build:   b,
		Commit:  commit,
		RepoDir: cloneDir,
	}, nil
}

func (c *cheopsImpl) Execute(ctxt *types.BuildContext) error {
	log.WithFields(log.Fields{
		"repo":   ctxt.Commit.RepoURL,
		"branch": ctxt.Commit.Branch,
		"commit": ctxt.Commit.ID,
	}).Debug("Executing task")

	for _, container := range ctxt.Build.Containers {
		log.WithFields(log.Fields{
			"container": container.Tag,
		}).Debug("Building image")
		tags := []string{container.Tag}

		dockerfile := container.Dockerfile
		if dockerfile == "" {
			dockerfile = "Dockerfile"
		}

		context := container.Context
		if context == "" {
			context = "."
		}

		err := docker.BuildImage(ctxt.RepoDir, dockerfile, tags, container.Args)
		if err != nil {
			log.WithFields(log.Fields{
				"container": container.Tag,
				"error":     err,
			}).Debug("Error building image")
			return err
		}
	}

	for _, action := range ctxt.Build.Actions {
		err := c.procAction(action)
		if err != nil {
			log.WithFields(log.Fields{
				"action": action.Type,
				"err":    err,
			}).Debug("Error processing action")
			return err
		}
	}

	return nil
}
