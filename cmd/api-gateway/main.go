package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	gateway "SaaS_Pipeline/internal/api-gateway"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := gateway.New(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize gateway: %v", err)
	}

	if err := app.Run(ctx); err != nil {
		log.Fatalf("Gateway stopped with error: %v", err)
	}
}
