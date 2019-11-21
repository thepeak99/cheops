package cheops

import (
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
		buildCtxt, err := webhook(r.Body, r.Header)
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"endpoint": endpoint,
			}).Warn("Webhook processing failed")

			return
		}
		go c.Execute(buildCtxt)
	})
}
