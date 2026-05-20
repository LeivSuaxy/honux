package migrate

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Migration struct {
	Version   int
	Name      string
	Direction string
	Content   string
	Path      string
}

type ByVersion []Migration

func (b ByVersion) Len() int           { return len(b) }
func (b ByVersion) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByVersion) Less(i, j int) bool { return b[i].Version < b[j].Version }

func discoverMigrations(dir string) ([]Migration, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("Error reading dir %s: %w", dir, err)
	}

	var migrations []Migration
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		m, err := parseMigrationFile(filepath.Join(dir, file.Name()))
		if err != nil {
			log.Printf("Warning: ignoring file %s - %v", file.Name(), err)
			continue
		}
		migrations = append(migrations, m)
	}
	return migrations, nil
}

func parseMigrationFile(path string) (Migration, error) {
	base := filepath.Base(path)

	if !strings.HasSuffix(base, ".sql") {
		return Migration{}, fmt.Errorf("This is not and .sql file")
	}

	base = strings.TrimSuffix(base, ".sql")
	parts := strings.Split(base, "_")
	if len(parts) < 3 {
		return Migration{}, fmt.Errorf("Invalid format, expected format: VERSION_NAME_DIRECTION")
	}

	direction := parts[len(parts)-1]
	if direction != "up" && direction != "down" {
		return Migration{}, fmt.Errorf("DIRECTION must be 'up' or 'down', direction obtained %s", direction)
	}

	versionStr := parts[0]
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return Migration{}, fmt.Errorf("Invalid version: %s", versionStr)
	}

	nameParts := parts[1 : len(parts)-1]
	name := strings.Join(nameParts, "_")

	content, err := os.ReadFile(path)
	if err != nil {
		return Migration{}, fmt.Errorf("Error reading File: %w", err)
	}

	return Migration{
		Version:   version,
		Name:      name,
		Direction: direction,
		Content:   string(content),
		Path:      path,
	}, nil
}

func groupByVersion(migrations []Migration) map[int]struct {
	Up   *Migration
	Down *Migration
} {
	group := make(map[int]struct {
		Up   *Migration
		Down *Migration
	})
	for i := range migrations {
		m := &migrations[i]
		entry := group[m.Version]
		if m.Direction == "up" {
			entry.Up = m
		} else {
			entry.Down = m
		}
		group[m.Version] = entry
	}
	return group
}

func groupByName(migrations []Migration) map[string]struct {
	Up   *Migration
	Down *Migration
} {
	group := make(map[string]struct {
		Up   *Migration
		Down *Migration
	})
	for i := range migrations {
		m := &migrations[i]
		entry := group[m.Name]
		if m.Direction == "up" {
			entry.Up = m
		} else {
			entry.Down = m
		}
		group[m.Name] = entry
	}
	return group
}

func getAppliedMigrations(db *sql.DB) (map[string]bool, error) {
	applied := make(map[string]bool)

	// Verificar si la tabla migration existe
	var tableExists bool
	err := db.QueryRow(`
        SELECT EXISTS (
            SELECT 1 FROM information_schema.tables
            WHERE table_name = 'migration'
        )
    `).Scan(&tableExists)
	if err != nil {
		return nil, err
	}
	if !tableExists {
		return applied, nil // tabla aún no creada, no hay migraciones aplicadas
	}

	rows, err := db.Query(`SELECT name FROM migration`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}
	return applied, nil
}

func applyUpMigration(db *sql.DB, m Migration) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(m.Content); err != nil {
		return fmt.Errorf("Error executing up %s: %w", m.Path, err)
	}

	_, err = tx.Exec(`INSERT INTO migration (name) VALUES ($1)`, m.Name)
	if err != nil {
		return fmt.Errorf("Error register migration %s: %w", m.Name, err)
	}

	return tx.Commit()
}

func applyDownMigration(db *sql.DB, m Migration) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(m.Content); err != nil {
		return fmt.Errorf("error ejecutando down %s: %w", m.Path, err)
	}
	_, err = tx.Exec(`DELETE FROM migration WHERE name = $1`, m.Name)
	if err != nil {
		return fmt.Errorf("error eliminando registro de migración %s: %w", m.Name, err)
	}
	return tx.Commit()
}

func MigrateUp(db *sql.DB, migrationsDir string) error {
	allMigrations, err := discoverMigrations(migrationsDir)
	if err != nil {
		return err
	}
	group := groupByName(allMigrations)

	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	var migrationsSorted []Migration
	for _, g := range group {
		if g.Up != nil {
			migrationsSorted = append(migrationsSorted, *g.Up)
		}
	}
	sort.Sort(ByVersion(migrationsSorted))

	for _, m := range migrationsSorted {
		if applied[m.Name] {
			fmt.Printf("Skipping %s (already applied)\n", m.Name)
			continue
		}
		fmt.Printf("Applying migration up: %s\n", m.Name)
		if err := applyUpMigration(db, m); err != nil {
			return err
		}
	}
	fmt.Println("Migrations up completed.")
	return nil
}

func MigrateDown(db *sql.DB, migrationsDir string) error {
	allMigrations, err := discoverMigrations(migrationsDir)
	if err != nil {
		return err
	}
	group := groupByName(allMigrations)

	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	if len(applied) == 0 {
		fmt.Println("There is not migrations to revert.")
		return nil
	}

	var lastApplied *Migration
	for name := range applied {
		if g, ok := group[name]; ok && g.Down != nil {
			candidate := g.Down
			if lastApplied == nil || candidate.Version > lastApplied.Version {
				lastApplied = candidate
			}
		}
	}

	if lastApplied == nil {
		return fmt.Errorf("Don't found down file for last applied migration.")
	}

	fmt.Printf("Reverting: %s\n", lastApplied.Name)
	if err := applyDownMigration(db, *lastApplied); err != nil {
		return err
	}
	fmt.Printf("Migration %s reverted.\n", lastApplied.Name)
	return nil
}

func MigrateToVersion(db *sql.DB, migrationDir string, targetVersion int) error {
	allMigrations, err := discoverMigrations(migrationDir)
	if err != nil {
		return err
	}
	group := groupByName(allMigrations)

	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	var allUp []Migration
	for _, g := range group {
		if g.Up != nil {
			allUp = append(allUp, *g.Up)
		}
	}
	sort.Sort(ByVersion(allUp))

	currentVersion := 0
	for name := range applied {
		if g, ok := group[name]; ok && g.Up != nil {
			if g.Up.Version > currentVersion {
				currentVersion = g.Up.Version
			}
		}
	}

	if targetVersion > currentVersion {
		for _, m := range allUp {
			if m.Version <= currentVersion {
				continue
			}

			if m.Version > targetVersion {
				break
			}

			if applied[m.Name] {
				continue
			}

			fmt.Printf("Applying up: %s\n", m.Name)
			if err := applyUpMigration(db, m); err != nil {
				return err
			}
		}
	} else if targetVersion < currentVersion {
		for i := len(allUp) - 1; i >= 0; i-- {
			m := allUp[i]
			if m.Version > currentVersion {
				continue
			}

			if m.Version <= targetVersion {
				break
			}

			if !applied[m.Name] {
				continue
			}

			down := group[m.Name].Down
			if down == nil {
				return fmt.Errorf("Doesn't exists down for version %d (%s)", m.Version, m.Name)
			}

			fmt.Printf("Reverting: %s\n", down.Name)
			if err := applyDownMigration(db, *down); err != nil {
				return err
			}
		}
	} else {
		fmt.Printf("You are already on version %d\n", targetVersion)
		return nil
	}

	fmt.Printf("Current Database on version %d\n", targetVersion)
	return nil
}
