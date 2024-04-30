package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/charlieegan3/storage-console/pkg/config"
	"github.com/charlieegan3/storage-console/pkg/server"
)

func main() {

	if len(os.Args) != 2 {
		log.Fatal("Please provide config as first arg")
	}

	configFile, err := os.OpenFile(os.Args[1], os.O_RDONLY, 0644)
	if err != nil {
		log.Fatalf("error reading config file: %v", err)
	}

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("error parsing config: %v", err)
	}

	srv, err := server.NewServer(cfg)
	if err != nil {
		log.Fatalf("error creating server: %v", err)
	}

	if logger := cfg.Server.LoggerInfo; logger != nil {
		logger.Printf(
			"Starting server on http://%s:%d\n",
			cfg.Server.Address,
			cfg.Server.Port,
		)
	}

	ctx := context.Background()

	err = srv.Start(ctx)
	if err != nil {
		log.Fatalf("error starting server: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("Received %v, shutting down...\n", sig)

		err = srv.Stop(ctx)
		if err != nil {
			log.Fatalf("error stopping server: %v", err)
		}

		os.Exit(0)
	}()

	log.Println("Press Ctrl+C to exit.")

	select {}
}
