package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"SaaS_Pipeline/internal/deploy"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	app, err := deploy.New(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	if err := app.Run(ctx); err != nil {
		log.Fatalf("App stopped with error: %v", err)
	}
}
