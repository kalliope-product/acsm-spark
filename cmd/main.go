package main

import (
	"context"
	"os"

	"github.com/kalliope-product/acsm-spark/internal/proxy"

	"github.com/hashicorp/go-hclog"
)

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "acsm-spark",
		Level: hclog.Debug,
	})

	logger.Info("Starting Spark Connect Proxy")
	sc := proxy.NewACSMServer(&proxy.ACSMServerOpts{
		Port:            15003,
		StaticProxyAddr: "localhost:15002",
	}, logger)

	err := sc.Start(context.Background())
	if err != nil {
		logger.Error("Failed to start Spark Connect Proxy", "error", err)
		os.Exit(1)
	}

}
