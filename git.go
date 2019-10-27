package cheops

import (
	"io/ioutil"
	"os"

	"gopkg.in/src-d/go-git.v4"
)

func cloneRepo(repoURL string) (string, error) {
	cloneDir, err := ioutil.TempDir("/tmp", "cheops")
	if err != nil {
		return "", err
	}

	_, err = git.PlainClone(cloneDir, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
	})
	if err != nil {
		return "", err
	}

	return cloneDir, nil
}
