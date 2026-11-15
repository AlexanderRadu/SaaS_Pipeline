package main

import (
	"SaaS_Pipeline/DataBase/migrations"
	"SaaS_Pipeline/pkg/config"
	"SaaS_Pipeline/pkg/migrator"
	"SaaS_Pipeline/pkg/postgres"
	"context"
	"embed"
	"log"

	"github.com/jackc/pgx/v5/stdlib"
)

func main() {
	config.LoadDotEnv(".env")
	dsn := config.MustGet("DATABASE_URL")
	serviceName := config.MustGet("SERVICE_NAME")

	var embedFS embed.FS

	var migrationsDir string

	switch serviceName {
	case "auth":

		migrationsDir = "auth"
		embedFS = migrations.AuthFS
	case "status":
		migrationsDir = "status"
		embedFS = migrations.StatusFS
	case "catalog":
		migrationsDir = "catalog"
		embedFS = migrations.CatalogFS
	default:
		log.Fatalf("unsupported or missing SERVICE_NAME: %q", serviceName)
	}

	ctx := context.Background()

	pool, err := postgres.NewPool(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}

	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	m, err := migrator.EmbedMigrations(db, embedFS, migrationsDir)
	if err != nil {
		log.Fatal(err)
	}

	if err := m.Up(); err != nil {
		log.Fatal(err)
	}
	log.Println("Migrations applied successfully")
}
