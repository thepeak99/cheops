package github

import (
	"bytes"
	"cheops/config"
	"cheops/git"
	"cheops/types"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

type GithubGitProvider struct {
	token    string
	cheops   types.Cheops
	endpoint string
	name     string
}

type githubPayload struct {
	Ref        string
	Repository struct {
		URL string
	}
	HeadCommit struct {
		Id string
	} `json:"head_commit"`
}

func New(cheops types.Cheops, providerConfig *config.GitProvider) (*GithubGitProvider, error) {
	log.WithFields(log.Fields{
		"provider": "Github",
	}).Debug("Initializing Git provider")

	endpoint := "/" + providerConfig.Name

	p := GithubGitProvider{
		token:    providerConfig.Token,
		cheops:   cheops,
		endpoint: endpoint,
		name:     providerConfig.Name,
	}

	cheops.RegisterWebhook(endpoint, func(body io.ReadCloser, headers map[string][]string) (*types.CommitInfo, error) {
		log.WithFields(log.Fields{
			"provider": providerConfig.Name,
		}).Debug("Received webhook")

		data, err := ioutil.ReadAll(body)
		if err != nil {
			return nil, err
		}

		log.WithFields(log.Fields{
			"provider": providerConfig.Name,
			"headers":  headers,
			"data":     string(data),
		}).Debug("Parsing webhook")

		event, ok := headers["X-Github-Event"]
		if !ok {
			return nil, errors.New("Failed to parse webhook, X-Github-Event header missing")
		}
		if event[0] != "push" {
			return nil, errors.New("Not a push event")
		}

		var payload githubPayload
		err = json.Unmarshal(data, &payload)
		if err != nil {
			log.WithFields(log.Fields{
				"provider": providerConfig.Name,
				"error":    err,
			}).Debug("Failed to parse webhook")
			return nil, err
		}

		var branch string
		refParts := strings.Split(payload.Ref, "/")
		if refParts[1] == "heads" {
			branch = refParts[2]
		} else {
			return nil, errors.New("Not a branch commit")
		}

		info := types.CommitInfo{
			ID:      payload.HeadCommit.Id,
			RepoURL: payload.Repository.URL,
			Branch:  branch,
		}

		return &info, nil
	})

	return &p, nil
}

func (p *GithubGitProvider) Clone(commit *types.CommitInfo, targetDir string) error {
	err := git.CloneRepoWithToken(commit.RepoURL, p.token, commit.ID, targetDir)
	if err != nil {
		return err
	}
	return nil
}

func (p *GithubGitProvider) RegisterRepo(repo *config.Repository) error {
	if !strings.HasPrefix(repo.URL, "https://github.com/") {
		return errors.New("The repository URL must start with https://github.com/")
	}

	if !strings.HasSuffix(repo.URL, ".git") {
		return errors.New("The repository URL must end with .git")
	}

	githubURL := "https://api.github.com/repos/" + repo.URL[19:len(repo.URL)-4] + "/hooks"

	webhookURL := p.cheops.Config().General.WebhookURL + p.endpoint

	body := map[string]interface{}{
		"name":   "web",
		"active": true,
		"events": []string{"push"},
		"config": map[string]interface{}{
			"url":          webhookURL,
			"content_type": "json",
		},
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return err
	}

	bodyReader := bytes.NewReader(bodyJSON)
	req, err := http.NewRequest(http.MethodPost, githubURL, bodyReader)
	if err != nil {
		return err
	}

	req.SetBasicAuth("user", p.token)

	log.WithFields(log.Fields{
		"repository": repo.URL,
		"provider":   p.name,
		"webhook":    webhookURL,
	}).Debug("Registering Github webhook")

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	switch res.StatusCode {
	case http.StatusCreated:
		return nil

	case http.StatusUnprocessableEntity:
		resBody, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return errors.New("Can't register webhook: " + res.Status)
		}

		resBodyJSON := make(map[string]interface{})
		err = json.Unmarshal(resBody, &resBodyJSON)
		if err != nil {
			return errors.New("Can't register webhook: " + res.Status)
		}

		resErrors, ok := resBodyJSON["errors"].([]interface{})
		if !ok {
			return errors.New("Can't register webhook: " + res.Status)
		}

		message, ok := resErrors[0].(map[string]interface{})["message"].(string)
		if ok && message == "Hook already exists on this repository" {
			return nil
		}
		return errors.New("Can't register webhook: " + res.Status)

	default:
		return errors.New("Can't register webhook: " + res.Status)
	}
}
