package main

import (
	"honux-core/internal/db"
	"honux-core/internal/db/migrate"
	"honux-core/internal/utils"
	"log"
	"os"
	"strconv"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cfg := utils.GetConfig()

	database, err := db.GetDb(cfg.DatabaseURL)

	if err != nil {
		log.Fatal("Error initializing database %w", err)
	}

	migrationsDir := "./internal/db/migrations"

	if len(os.Args) < 2 {
		log.Fatal("Use: migrate [up|down|to] N")
	}

	switch os.Args[1] {
	case "up":
		if err := migrate.MigrateUp(database.DB, migrationsDir); err != nil {
			log.Fatal(err)
		}
	case "down":
		if err := migrate.MigrateDown(database.DB, migrationsDir); err != nil {
			log.Fatal(err)
		}
	case "to":
		if len(os.Args) != 3 {
			log.Fatal("Uso: migrate to <version>")
		}
		target, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatal("Versión inválida")
		}
		if err := migrate.MigrateToVersion(database.DB, migrationsDir, target); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("Comando no reconocido")
	}
}
