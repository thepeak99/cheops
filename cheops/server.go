package cheops

import (
	"cheops/config"
	"cheops/types"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func (c *cheopsImpl) Serve() error {
	general := c.Config().General
	log.WithFields(log.Fields{
		"bindAddr": general.BindAddr,
	}).Info("Webhook Server listening")

	return http.ListenAndServeTLS(general.BindAddr, general.TLSCert, general.TLSKey, nil)
}

func (c *cheopsImpl) RegisterWebhook(endpoint string, webhook types.WebhookFunc) {
	log.WithFields(log.Fields{
		"endpoint": endpoint,
	}).Debug("Registering webhook")

	http.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(log.Fields{
			"endpoint": endpoint,
		}).Debug("Received webhook")

		w.WriteHeader(200)
		commit, err := webhook(r.Body, r.Header)
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"endpoint": endpoint,
			}).Warn("Webhook processing failed")

			return
		}

		var repo *config.Repository
		for _, repo = range c.Config().Repos {
			if repo.URL == commit.RepoURL {
				break
			}
		}

		if repo == nil {
			log.WithFields(log.Fields{
				"endpoint":   endpoint,
				"repository": commit.RepoURL,
				"branch":     commit.Branch,
			}).Warn("Not building unknown repo")
			return
		}

		if repo.Branch != commit.Branch {
			log.WithFields(log.Fields{
				"endpoint": endpoint,
				"branch":   commit.Branch,
			}).Debug("Not building unknown branch")

			return
		}

		go func() {
			ctxt, err := c.GetBuildContext(repo, commit)
			if err != nil {
				return
			}
			c.Execute(ctxt)
		}()
	})
}
