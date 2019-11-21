package git

import (
	"os"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

func checkoutToCommit(repo *git.Repository, commit string) error {
	tree, err := repo.Worktree()
	if err != nil {
		return err
	}

	err = tree.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(commit),
	})
	if err != nil {
		return err
	}

	return nil
}

func CloneRepo(repoURL, commit, targetDir string) error {
	repo, err := git.PlainClone(targetDir, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
	})
	if err != nil {
		return err
	}

	return checkoutToCommit(repo, commit)
}

func CloneRepoWithToken(repoURL, token, commit, targetDir string) error {
	repo, err := git.PlainClone(targetDir, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
		Auth: &http.BasicAuth{
			Username: "token",
			Password: token,
		},
	})
	if err != nil {
		return err
	}

	return checkoutToCommit(repo, commit)
}
