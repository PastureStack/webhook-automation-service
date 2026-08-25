package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/PastureStack/webhook-automation-service/drivers"
	"github.com/PastureStack/webhook-automation-service/service"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

var VERSION = "0.0.0"

func main() {
	app := &cli.Command{
		Name:    "webhook-automation-service",
		Version: VERSION,
		Usage:   "Run webhook-driven automation for the PastureStack control plane",
		Action:  StartWebhook,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "public-key-file",
				Aliases: []string{"rsa-public-key-file"},
				Usage:   "path to the RSA public key used to verify webhook tokens",
				Sources: cli.EnvVars("PASTURESTACK_API_PUBLIC_KEY_FILE"),
			},
			&cli.StringFlag{
				Name:    "public-key-contents",
				Aliases: []string{"rsa-public-key-contents"},
				Usage:   "PEM-encoded RSA public key; an alternative to --public-key-file",
				Sources: cli.EnvVars("PASTURESTACK_API_PUBLIC_KEY_CONTENTS", "RSA_PUBLIC_KEY_CONTENTS"),
			},
			&cli.StringFlag{
				Name:    "listen-address",
				Usage:   "HTTP listen address",
				Value:   "127.0.0.1:8085",
				Sources: cli.EnvVars("PASTURESTACK_WEBHOOK_LISTEN_ADDRESS"),
			},
		},
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func StartWebhook(_ context.Context, c *cli.Command) error {
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
