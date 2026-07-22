package main

import (
	"net/http"
	"os"
	"time"

	"github.com/PastureStack/webhook-automation-service/drivers"
	"github.com/PastureStack/webhook-automation-service/service"
	log "github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
)

var VERSION = "0.0.0"

func main() {
	app := cli.NewApp()
	app.Name = "webhook-automation-service"
	app.Version = VERSION
	app.Usage = "Run webhook-driven automation for the PastureStack control plane"
	app.Action = StartWebhook
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "public-key-file, rsa-public-key-file",
			Usage:  "path to the RSA public key used to verify webhook tokens",
			EnvVar: "PASTURESTACK_API_PUBLIC_KEY_FILE",
		},
		cli.StringFlag{
			Name:   "public-key-contents, rsa-public-key-contents",
			Usage:  "PEM-encoded RSA public key; an alternative to --public-key-file",
			EnvVar: "PASTURESTACK_API_PUBLIC_KEY_CONTENTS,RSA_PUBLIC_KEY_CONTENTS",
		},
		cli.StringFlag{
			Name:   "listen-address",
			Usage:  "HTTP listen address",
			Value:  "127.0.0.1:8085",
			EnvVar: "PASTURESTACK_WEBHOOK_LISTEN_ADDRESS",
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func StartWebhook(c *cli.Context) error {
	drivers.RegisterDrivers()
	publicKey, err := service.GetPublicKey(c)
	if err != nil {
		return err
	}

	rh := &service.RouteHandler{
		PublicKey:     publicKey,
		ClientFactory: &service.ClientFactory{},
	}
	router := service.NewRouter(rh)
	address := c.String("listen-address")
	server := &http.Server{
		Addr:              address,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	log.Infof("Webhook automation service listening on %s", address)
	return server.ListenAndServe()
}
