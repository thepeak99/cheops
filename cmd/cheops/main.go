package main

import (
	"cheops/cheops"

	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetLevel(log.DebugLevel)

	log.Info("Starting Cheops")

	cheops := cheops.New()
	if err := cheops.Serve(); err != nil {
		log.WithFields(log.Fields{
			"bindAddr": cheops.Config().General.BindAddr,
			"error":    err,
		}).Fatal("Can't listen on port")
	}
}
