package main

import (
	"SaaS_Pipeline/DataBase/migrations"
	"SaaS_Pipeline/pkg/config"
	"SaaS_Pipeline/pkg/migrator"
	"SaaS_Pipeline/pkg/postgres"
	"context"
	"log"

	"github.com/jackc/pgx/v5/stdlib"
)

func main() {
	config.LoadDotEnv(".env")
	dsn := config.MustGet("DATABASE_URL")

	ctx := context.Background()

	pool, err := postgres.NewPool(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}

	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	m, err := migrator.EmbedMigrations(db, migrations.FS, ".")
	if err != nil {
		log.Fatal(err)
	}

	if err := m.Up(); err != nil {
		log.Fatal(err)
	}
	log.Println("Migrations applied successfully")
}
