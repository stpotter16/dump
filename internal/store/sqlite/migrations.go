package sqlite

import (
	"cmp"
	"context"
	"embed"
	"fmt"
	"log"
	"path"
	"slices"
	"strconv"
	"time"
)

type migration struct {
	version   int
	statement string
}

//go:embed migrations/*.sql
var migrationFs embed.FS

func (s Store) runMigrations() error {
	var currentSchemaVersion int
	pragmaCtx, pragmaCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pragmaCancel()
	row := s.db.QueryRow(pragmaCtx, `PRAGMA user_version`)
	if err := row.Scan(&currentSchemaVersion); err != nil {
		log.Printf("Could not read the current schema version: %v", err)
		return err
	}

	migrations, err := loadMigrations()
	if err != nil {
		log.Printf("Could not load database migrations: %v", err)
		return err
	}

	log.Printf("Current database schema version is %d out of %d migrations", currentSchemaVersion, len(migrations))

	for _, migration := range migrations {
		if currentSchemaVersion >= migration.version {
			continue
		}

		transactions := []string{
			migration.statement,
			fmt.Sprintf(`PRAGMA user_version=%d`, migration.version),
		}

		migrationCtx, migrationCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer migrationCancel()

		if err := s.db.ExecuteTransaction(migrationCtx, transactions...); err != nil {
			log.Printf("Could not execute migration %d: %v", migration.version, err)
			return err
		}

		log.Printf("Current database version is %d out of %d migrations", migration.version, len(migrations))
	}

	return nil
}

func loadMigrations() ([]migration, error) {
	migrations := []migration{}

	migrationsDir := "migrations"

	migrationEntries, err := migrationFs.ReadDir(migrationsDir)
	if err != nil {
		return []migration{}, err
	}

	for _, entry := range migrationEntries {
		if entry.IsDir() {
			continue
		}

		version, err := parseVersionNumber(entry.Name())
		if err != nil {
			return []migration{}, err
		}

		statement, err := migrationFs.ReadFile(path.Join(migrationsDir, entry.Name()))
		if err != nil {
			return []migration{}, err
		}

		migrations = append(migrations, migration{version, string(statement)})
	}

	slices.SortFunc(migrations,
		func(a, b migration) int {
			return cmp.Compare(a.version, b.version)
		})

	return migrations, nil
}

func parseVersionNumber(name string) (int, error) {
	version, err := strconv.ParseInt(name[:3], 10, 32)
	if err != nil {
		return 0, err
	}

	return int(version), nil
}
