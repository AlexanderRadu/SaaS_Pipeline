package migrations

import "embed"

//go:embed auth/*.sql
var AuthFS embed.FS

//go:embed catalog/*.sql
var CatalogFS embed.FS

//go:embed status/*.sql
var StatusFS embed.FS
