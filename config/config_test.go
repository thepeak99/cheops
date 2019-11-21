package config

import (
	"testing"
)

var sampleConfig = `
general:
  webhook_url: https://cheops.io/

builds:
  - name: XX
    repo:
      type: github
      url: https://github.com/patata/patat.git
      branch: branch
    containers:
      - dockerfile: tato
        context: .
        name: container-1
        args:
          ARG1: patata

      - dockerfile: bla
        context: tato
    actions:
      - type: push
        container: container-1
        repo: 1848484.amazon.com/repo
        provider: aws
      - exec: container-2
        commands:
          - tato
          - ls 
    notifiers:
      - type: github
      - type: generic
        url: tato
      
providers:
  git:
    - name: github
      token: bla
      type: github
    - name: bitbucket
      token: tato
      type: bitbucket

  docker_creds:
    - name: aws
      type: aws
      secret_id: bla
`

func TestParseConfig(t *testing.T) {
	config, err := parseConfig([]byte(sampleConfig))

	if err != nil {
		t.Error(err)
	}

	if config.Builds[0].Name != "XX" {
		t.Error("uhm")
	}

	if *config.Builds[0].Containers[0].Args["ARG1"] != "patata" {
		t.Error("Fail")
	}

	if config.General.WebhookURL != "https://cheops.io/" {
		t.Error("Fail")
	}
}
