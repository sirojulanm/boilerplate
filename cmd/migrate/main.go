package main

import (
	"os"
	"time"

	"github.com/sirojulanam/boilerplate/internal/config"
	"github.com/sirojulanam/boilerplate/internal/helper"
	"github.com/sirojulanam/boilerplate/internal/infrastructure/postgres"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339})
	log.SetOutput(os.Stdout)

	if err := helper.LoadEnvFile(helper.Env("ENV_FILE", ".env")); err != nil {
		log.WithError(err).Error("load environment file")
		os.Exit(1)
	}

	database, err := config.LoadDatabase()
	if err != nil {
		log.WithError(err).Error("load database configuration")
		os.Exit(1)
	}

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	switch command {
	case "up":
		if err := postgres.MigrateUp(database.URL); err != nil {
			log.WithError(err).Error("migrate up failed")
			os.Exit(1)
		}
		log.WithField("database", database.Name).Info("migrations applied")
	case "down":
		if err := postgres.MigrateDown(database.URL); err != nil {
			log.WithError(err).Error("migrate down failed")
			os.Exit(1)
		}
		log.WithField("database", database.Name).Info("migration rolled back")
	case "version":
		version, dirty, err := postgres.MigrationVersion(database.URL)
		if err != nil {
			log.WithError(err).Error("read migration version failed")
			os.Exit(1)
		}
		log.WithFields(logrus.Fields{
			"database": database.Name,
			"version":  version,
			"dirty":    dirty,
		}).Info("migration version")
	default:
		log.WithField("command", command).Error("unknown command; use up, down, or version")
		os.Exit(1)
	}
}
