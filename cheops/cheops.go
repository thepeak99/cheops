package cheops

import (
	"cheops/aws"
	"cheops/config"
	"cheops/docker"
	"cheops/github"
	"cheops/types"
	"errors"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
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
		c.gitProviders[gitProvider.Name], err = c.initGitProvider(&gitProvider)
		if err != nil {
			log.WithFields(log.Fields{
				"provider": gitProvider.Name,
				"error":    err,
			}).Fatal("Error loading provider")
		}
	}

	log.Debug("Initializing Docker Credential providers")
	for _, dockerCredsProvider := range config.Providers.DockerCreds {
		c.dockerCredsProviders[dockerCredsProvider.Name], err = c.initDockerCredsProvider(&dockerCredsProvider)
		if err != nil {
			log.WithFields(log.Fields{
				"provider": dockerCredsProvider.Name,
				"error":    err,
			}).Fatal("Error loading provider")
		}
	}

	log.Debug("Loading builds")
	for _, build := range config.Builds {
		provider, ok := c.gitProviders[build.Repo.Provider]
		if !ok {
			log.WithFields(log.Fields{
				"build":    build.Name,
				"provider": build.Repo.Provider,
			}).Fatal("Unknown Git provider")
		}

		err := provider.RegisterRepo(&build.Repo)
		if err != nil {
			log.WithFields(log.Fields{
				"build":    build.Name,
				"provider": build.Repo.Provider,
				"repo":     build.Repo.URL,
				"error":    err,
			}).Fatal("Can't register repository to provider")
		}
	}

	return &c
}

func (c *cheopsImpl) procAction(action *config.Action) error {
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

		err = docker.PushImage(action.Tag, creds)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *cheopsImpl) Execute(ctxt *types.BuildContext) error {
	log.WithFields(log.Fields{
		"build":  ctxt.Build.Name,
		"branch": ctxt.Branch,
		"commit": ctxt.Commit,
	}).Debug("Executing task")

	cloneDir, err := ioutil.TempDir("/tmp", "cheops")
	if err != nil {
		return err
	}

	provider := c.gitProviders[ctxt.Build.Repo.Provider]
	err = provider.Clone(&ctxt.Build.Repo, ctxt.Commit, cloneDir)
	if err != nil {
		return err
	}

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

		err := docker.BuildImage(cloneDir, dockerfile, tags, container.Args)
		if err != nil {
			log.WithFields(log.Fields{
				"container": container.Tag,
				"error":     err,
			}).Debug("Error building image")
			return err
		}
	}

	for _, action := range ctxt.Build.Actions {
		err := c.procAction(&action)
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
