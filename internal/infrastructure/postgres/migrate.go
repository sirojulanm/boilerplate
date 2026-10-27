package postgres

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	pgxdriver "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sirojulanam/boilerplate/migrations"
)

// MigrateUp applies every pending migration.
func MigrateUp(dsn string) (err error) {
	migrator, err := newMigrator(dsn)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := closeMigrator(migrator); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// MigrateDown rolls back exactly one migration.
func MigrateDown(dsn string) (err error) {
	migrator, err := newMigrator(dsn)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := closeMigrator(migrator); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if err := migrator.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("roll back migration: %w", err)
	}
	return nil
}

// MigrationVersion reports the current schema version and dirty flag.
func MigrationVersion(dsn string) (version uint, dirty bool, err error) {
	migrator, err := newMigrator(dsn)
	if err != nil {
		return 0, false, err
	}
	defer func() {
		if closeErr := closeMigrator(migrator); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	version, dirty, err = migrator.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("read migration version: %w", err)
	}
	return version, dirty, nil
}

func newMigrator(dsn string) (*migrate.Migrate, error) {
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("load embedded migrations: %w", err)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open migration database: %w", err)
	}

	driver, err := pgxdriver.WithInstance(db, &pgxdriver.Config{})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create migration driver: %w", err)
	}

	migrator, err := migrate.NewWithInstance("iofs", source, "pgx5", driver)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create migrator: %w", err)
	}
	return migrator, nil
}

func closeMigrator(migrator *migrate.Migrate) error {
	sourceErr, databaseErr := migrator.Close()
	return errors.Join(sourceErr, databaseErr)
}
