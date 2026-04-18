package main

import (
	"SaaS_Pipeline/internal/status"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := status.New(ctx)
	if err != nil {
		log.Fatal("Failed to initialize app: %v", err)
	}
	if err := app.Run(ctx); err != nil {
		log.Fatal("Failed to run app: %v", err)
	}
}
