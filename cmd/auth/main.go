package main

import (
	"SaaS_Pipeline/internal/auth/app"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := app.New(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}
	if err := app.Run(ctx); err != nil {
		log.Fatalf("Failed to run app: %v", err)
	}
}
